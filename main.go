package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	uploadBufferSize = 26 * 1024 * 1024
)

var (
	reFastComScript = regexp.MustCompile(`(?m)<script\s+src="\/(\S+)">`)
	reFastComToken  = regexp.MustCompile(`(?mU)token:"(\S+)"`)

	commit  = ""
	version = "dev"

	parallelConnections = 5

	downloadDuration = 30 * time.Second
	uploadDuration   = 30 * time.Second
)

type APIResponse struct {
	Client struct {
		IP       string `json:"ip"`
		Asn      string `json:"asn"`
		Location struct {
			City    string `json:"city"`
			Country string `json:"country"`
		} `json:"location"`
	} `json:"client"`
	Targets []Target `json:"targets"`
}

type Target struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Location struct {
		City    string `json:"city"`
		Country string `json:"country"`
	} `json:"location"`
}

type TestResult struct {
	TotalSize uint64
	TotalTime time.Duration
	Target    Target
}

func init() {
	// init logger
	if version == "dev" {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// read env vars parallelConnections
	if os.Getenv("PARALLEL_CONNECTIONS") != "" {
		i, err := strconv.Atoi(os.Getenv("PARALLEL_CONNECTIONS"))
		if err != nil {
			log.Fatal().Err(err).Msg("Error reading PARALLEL_CONNECTIONS")
		}
		parallelConnections = i
	}
	if os.Getenv("DOWNLOAD_DURATION_SEC") != "" {
		i, err := strconv.Atoi(os.Getenv("DOWNLOAD_DURATION_SEC"))
		if err != nil {
			log.Fatal().Err(err).Msg("Error reading DOWNLOAD_DURATION_SEC")
		}
		downloadDuration = time.Duration(i) * time.Second
	}
	if os.Getenv("UPLOAD_DURATION_SEC") != "" {
		i, err := strconv.Atoi(os.Getenv("UPLOAD_DURATION_SEC"))
		if err != nil {
			log.Fatal().Err(err).Msg("Error reading UPLOAD_DURATION_SEC")
		}
		uploadDuration = time.Duration(i) * time.Second
	}

}

func main() {

	log.Info().Str("version", version).Str("commit", commit).Msg("Starting fast-speedtest")
	log.Info().Int("parallelConnections", parallelConnections).Float64("downloadDurationSec", downloadDuration.Seconds()).Float64("uploadDurationSec", uploadDuration.Seconds()).Msg("Config")

	//startTest()
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		downloadSpeed, uploadSpeed := startTest()
		log.Info().Float64("downloadSpeed", downloadSpeed).Float64("uploadSpeed", uploadSpeed).Msg("Speedtest results")
		w.Header().Add("content-type", "text/plain")
		w.Write([]byte("# TYPE speedtest_bits_per_second gauge\n"))
		w.Write([]byte("# HELP speedtest_bits_per_second Speed measured against fast.com\n"))
		w.Write([]byte(`speedtest_bits_per_second{direction="downstream"} ` + fmt.Sprintf("%.0f", downloadSpeed*8) + "\n"))
		w.Write([]byte(`speedtest_bits_per_second{direction="upstream"} ` + fmt.Sprintf("%.0f", uploadSpeed*8) + "\n"))
	})

	log.Fatal().Err(http.ListenAndServe(":9696", nil))
}

func startTest() (float64, float64) {
	startTime := time.Now()
	defer func() { log.Debug().Dur("duration", time.Since(startTime)).Msg("Test duration") }()

	targets, err := getTargets()
	if err != nil {
		panic(err)
	}

	log.Debug().Any("targets", targets).Msg("Targets")

	downloadSpeed := startDownloadTest(targets)
	uploadSpeed := startUploadTest(targets)

	return downloadSpeed, uploadSpeed

	//
	//for i := 0; i < maxConnections-1; i++ {
	//	result := <-resultChan
	//	totalSize += result.TotalSize
	//	totalTime += result.TotalTime
	//}
	//
	//speed := (float64(totalSize * int64(time.Second))) / float64(totalTime/maxConnections)
	//speedDownloadMbs := speed / 1e3 / 1e3 * 8
	//
	//fmt.Printf("%s: %.2f Mbps\n", "Download", speedDownloadMbs)
	//
	//return speed
}
