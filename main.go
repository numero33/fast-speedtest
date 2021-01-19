package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const downloadBufferSize = 26 * 1024 * 1024
const maxDownloadDuration = 40 * time.Second
const minDownloadDuration = 30 * time.Second
const maxConnections = 5

var reFastComScript = regexp.MustCompile(`(?m)<script\s+src="\/(\S+)">`)
var reFastComToken = regexp.MustCompile(`(?mU)token:"(\S+)"`)

type APIResponse struct {
	Client struct {
		Asn      string      `json:"asn"`
		Isp      interface{} `json:"isp"`
		Location struct {
			Country string `json:"country"`
			City    string `json:"city"`
		} `json:"location"`
		IP string `json:"ip"`
	} `json:"client"`
	Targets []struct {
		URL      string `json:"url"`
		Location struct {
			Country string `json:"country"`
			City    string `json:"city"`
		} `json:"location"`
		Name string `json:"name"`
	} `json:"targets"`
}

type TestResult struct {
	TotalSize int64
	TotalTime time.Duration
}

func main() {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		speed := startTest()
		w.Header().Add("content-type", "text/plain")
		w.Write([]byte("# TYPE speedtest_bits_per_second gauge\n"))
		w.Write([]byte("# HELP speedtest_bits_per_second Speed measured against fast.com\n"))
		w.Write([]byte(`speedtest_bits_per_second{direction="downstream"} ` + fmt.Sprintf("%.0f", speed*8) + "\n"))
	})

	log.Fatal(http.ListenAndServe(":9696", nil))
}

func startTest() float64 {
	resp, err := http.Get("https://fast.com")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	scriptFound := reFastComScript.FindStringSubmatch(string(b))
	fmt.Println(scriptFound[1])

	resp, err = http.Get("https://fast.com/" + scriptFound[1])
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	tokenFound := reFastComToken.FindStringSubmatch(string(b))
	fmt.Println(tokenFound[1])

	resp, err = http.Get("https://api.fast.com/netflix/speedtest/v2?https=true&token=" + tokenFound[1] + "&urlCount=" + strconv.Itoa(maxConnections))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var response APIResponse

	if err := decoder.Decode(&response); err != nil {
		panic(err)
	}

	start := time.Now()
	resultChan := make(chan TestResult, maxConnections)

	fmt.Println(start)

	for i := 0; i <= maxConnections-1; i++ {
		go startDownloadConnection(response.Targets[0].URL, start, resultChan)
	}

	var totalSize int64 = 0
	var totalTime time.Duration

	for i := 0; i <= maxConnections-1; i++ {
		result := <-resultChan
		totalSize += result.TotalSize
		totalTime += result.TotalTime
	}

	speed := (float64(totalSize * int64(time.Second))) / float64(totalTime/maxConnections)
	speedDownloadMbs := speed / 1e3 / 1e3 * 8

	fmt.Printf("%s: %.2f Mbps\n", "Download", speedDownloadMbs)

	return speed
}

func startDownloadConnection(url string, start time.Time, result chan TestResult) {
	fmt.Println("startDownloadConnection")
	resultChan := make(chan TestResult)
	sumResult := TestResult{}

	defer func() {
		result <- sumResult
	}()
	for time.Since(start) <= minDownloadDuration {
		go downloadTest(url, start, resultChan)
		result := <-resultChan
		sumResult.TotalSize += result.TotalSize
		sumResult.TotalTime += result.TotalTime
	}
}

func downloadTest(url string, start time.Time, result chan TestResult) {
	var totalRead int64 = 0
	startTest := time.Now()
	defer func() {
		result <- TestResult{
			TotalSize: totalRead,
			TotalTime: time.Since(startTest),
		}
	}()

	url = strings.Replace(url, "speedtest", "speedtest/range/0-"+strconv.Itoa(downloadBufferSize), 1)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 100*1024)
	for time.Since(start) <= maxDownloadDuration {
		read, err := resp.Body.Read(buf)
		totalRead += int64(read)
		//fmt.Println(totalRead)
		if err != nil {
			if err != io.EOF {
				log.Printf("[%s] Download error: %v\n", url, err)
			}
			break
		}
	}
}
