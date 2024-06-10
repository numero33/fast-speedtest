package main

import (
	"context"
	"crypto/rand"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io"
	mrand "math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

func startUploadTest(targets []Target) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), uploadDuration)
	defer cancel()

	resultChan := make(chan *TestResult, parallelConnections)

	var wg sync.WaitGroup

	wg.Add(parallelConnections)
	for i := 0; i < parallelConnections; i++ {
		go func(ctx context.Context, targets []Target) {
			defer wg.Done()
			result, err := startUploadConnection(ctx, targets)
			if err != nil {
				log.Error().Err(err).Msg("Error")
				return
			}
			resultChan <- result
		}(ctx, targets)
	}

	var totalSize uint64 = 0
	var totalTime time.Duration

	wg.Wait()
	close(resultChan)

	for result := range resultChan {
		totalSize += result.TotalSize
		totalTime += result.TotalTime
	}

	speed := (float64(totalSize * uint64(time.Second))) / float64(totalTime) * float64(parallelConnections)

	log.Debug().Dur("totalTime", totalTime).Uint64("totalSize", totalSize).Float64("speed", speed/1e3/1e3*8).Msg("UploadSpeed")

	return speed
}

func startUploadConnection(ctx context.Context, targets []Target) (*TestResult, error) {
	log.Debug().Msg("startUploadConnection")

	// random target
	testResult := &TestResult{
		TotalSize: 0,
		TotalTime: 0,
		Target:    targets[mrand.Intn(len(targets))],
	}

	for {
		select {
		case <-ctx.Done():
			return testResult, nil
		default:
			url := strings.Replace(testResult.Target.URL, "/speedtest", "/speedtest/range/0-"+strconv.Itoa(uploadBufferSize), 1)

			log.Debug().Str("url", url).Msg("Upload")
			result, err := uploadTest(ctx, url)
			if err != nil {
				return nil, err
			}
			testResult.TotalSize += result.TotalSize
			testResult.TotalTime += result.TotalTime
			log.Debug().Str("url", url).Dur("spendtime", result.TotalTime).Uint64("uploadedSize", result.TotalSize).Msg("Uploaded")
		}
	}
}

func uploadTest(ctx context.Context, url string) (*TestResult, error) {

	written := uint64(0)
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		size := 1024
		buf := make([]byte, size)
		rand.Read(buf)
		for {
			//log.Debug().Msg("Reading")
			select {
			case <-ctx.Done():
				return
			default:
				wn, err := writer.Write(buf)
				if err != nil {
					panic(err)
				}
				written += uint64(wn)
				if wn != size {
					panic("short write")
				}
				if written >= uploadBufferSize {
					writer.Close()
					return
				}
			}
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, io.NopCloser(reader))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = uploadBufferSize

	startTime := time.Now()

	// skip error because of context timeout
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Error().Int("statusCode", resp.StatusCode).Msg("Request failed")
			return nil, errors.Errorf("Request failed with response code: %d", resp.StatusCode)
		}
	}

	return &TestResult{
		TotalSize: written,
		TotalTime: time.Since(startTime),
	}, nil
}
