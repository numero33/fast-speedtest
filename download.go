package main

import (
	"context"
	"github.com/rs/zerolog/log"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

func startDownloadTest(targets []Target) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), downloadDuration)
	defer cancel()

	resultChan := make(chan *TestResult, parallelConnections)

	var wg sync.WaitGroup

	wg.Add(parallelConnections)
	for i := 0; i < parallelConnections; i++ {
		go func(ctx context.Context, targets []Target) {
			defer wg.Done()
			result, err := startDownloadConnection(ctx, targets)
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

	downloadSpeed := (float64(totalSize * uint64(time.Second))) / float64(totalTime) * float64(parallelConnections)

	log.Debug().Dur("totalTime", totalTime).Uint64("totalSize", totalSize).Float64("speed", downloadSpeed/1e3/1e3*8).Msg("DownloadSpeed")

	return downloadSpeed
}

func startDownloadConnection(ctx context.Context, targets []Target) (*TestResult, error) {
	log.Debug().Msg("startDownloadConnection")

	// random target
	testResult := &TestResult{
		TotalSize: 0,
		TotalTime: 0,
		Target:    targets[rand.Intn(len(targets))],
	}

	for {
		select {
		case <-ctx.Done():
			return testResult, nil
		default:
			//url := strings.Replace(testResult.Target.URL, "/speedtest", "/speedtest/range/0-"+strconv.Itoa(size), 1)
			url := testResult.Target.URL

			log.Debug().Str("url", url).Msg("Downloading")
			result, err := downloadTest(ctx, url)
			if err != nil {
				return nil, err
			}
			testResult.TotalSize += result.TotalSize
			testResult.TotalTime += result.TotalTime
			log.Debug().Str("url", url).Dur("spendtime", result.TotalTime).Uint64("downloaded", result.TotalSize).Msg("Downloaded")

			//changeFact := downloadDuration.Seconds() / result.TotalTime.Seconds()
			//if changeFact > 2 || changeFact < 0.5 {
			//	log.Debug().Int("oldSize", size).Int("newSize", int(float64(size)*changeFact)).Float64("changeFact", changeFact).Msg("Changing size")
			//	size = int(float64(size) * changeFact)
			//}
		}
	}
}

func downloadTest(ctx context.Context, url string) (*TestResult, error) {
	startTime := time.Now()

	result := &TestResult{
		TotalSize: 0,
		TotalTime: 0,
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 100*1024)

	for {
		select {
		case <-ctx.Done():
			log.Debug().Str("url", url).Msg("Context done")
			return result, nil
		default:
			read, err := resp.Body.Read(buf)
			result.TotalTime = time.Since(startTime)
			result.TotalSize += uint64(read)
			if err != nil {
				if err == io.EOF {
					return result, nil
				}
				log.Error().Err(err).Msg("Error")
				return nil, err
			}

		}
	}
}
