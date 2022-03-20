package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	limitParallelReq      = 10
	defaultParallelReq    = 1
	requestTimeout        = 5 * time.Second
	tcpConnectTimeout     = 3 * time.Second
	tlsHandshakeTimeout   = 1 * time.Second
	responseHeaderTimeout = 1 * time.Second
)

var NoUrlsError = errors.New("No urls")

// init logger. Output for stderr
// flag "" means no prefix. flag "0" means no time in log.
// It's bad to create global logger, but i dont want to write a log of code with modules, because its simple tool
var logger = log.New(os.Stderr, "", 0)

func main() {
	// parse and validate input params
	countParallelReq := flag.Int("parallel", defaultParallelReq, "count of parallel requests")
	flag.Parse()

	urls := flag.Args()
	urls, err := getURLS(urls)
	if err != nil {
		return
	}
	parallelReqCount := getParallelCount(*countParallelReq, len(urls))
	// creating http client
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: tcpConnectTimeout,
		}).Dial,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		MaxIdleConns:          1,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
	}
	// run workers, that will process our requests
	res := Process(httpClient, parallelReqCount, urls)

	fmt.Println(res)
}

func getURLS(urls []string) ([]string, error) {
	result := make([]string, 0)
	if len(urls) == 0 {
		return result, NoUrlsError
	}
	storage := make(map[string]bool)
	for _, val := range urls {
		if !strings.HasPrefix(val, "http") {
			val = "http://" + val
		}
		_, err := url.ParseRequestURI(val)
		if err != nil {
			return result, err
		}
		if _, ok := storage[val]; !ok {
			result = append(result, val)
		}
		storage[val] = true
	}
	return result, nil
}

// getParallelCount try to validate count of parallel request
// and return correct value of parallel request count.
func getParallelCount(parallelCount, urlsCount int) int {
	if parallelCount < 1 {
		logger.Printf("Ignore required count of parallel requests %d because it's less then 1\n", parallelCount)
		parallelCount = defaultParallelReq
	}
	if parallelCount > limitParallelReq {
		logger.Printf("Ignore required count of parallel requests %d because it's more than limit %d\n", parallelCount, limitParallelReq)
		parallelCount = limitParallelReq
	}
	if parallelCount > urlsCount {
		logger.Printf("Ignore required count of parallel requests %d because it's more than count of urls %d\n", parallelCount, urlsCount)
		parallelCount = urlsCount
	}
	return parallelCount
}

// Process func run workers ( amount is equal to parallelCount ).
// Each worker can proccess some requests for urls.
// It blocks util all requests complete.
func Process(httpClient *http.Client, parallelCount int, urls []string) string {
	dataCh := make(chan string)
	var result bytes.Buffer

	var wg sync.WaitGroup
	for i := 0; i < parallelCount; i++ {
		wg.Add(1)
		go func(cl *http.Client, dCh <-chan string, wGroup *sync.WaitGroup) {
			defer wGroup.Done()
			for {
				url, ok := <-dCh
				if !ok {
					break
				}
				resp := Request(cl, url)
				if result.Len() != 0 {
					result.WriteString(" ")
				}
				result.WriteString(resp)
			}
		}(httpClient, dataCh, &wg)
	}
	for _, url := range urls {
		dataCh <- url
	}
	close(dataCh)
	wg.Wait()
	return result.String()
}

// Request function make http request for current url and return result.
// result is url + space + md5(from response body)
func Request(client *http.Client, url string) string {
	result := url
	resp, err := client.Get(url)
	if err != nil {
		if os.IsTimeout(err) {
			result += " error:timeout"
		} else {
			result += " error:unknown"
		}
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		result += " error:" + strconv.Itoa(resp.StatusCode)
		return result
	}
	buffer := make([]byte, md5.BlockSize*128, md5.BlockSize*128) // buffer size ~ 8KB ( 128 * 64B )
	hasher := md5.New()
	for {
		length, err := resp.Body.Read(buffer)
		if err != nil {
			if err == io.EOF {
				hasher.Write(buffer[0:length])
				result += fmt.Sprintf(" %x", hasher.Sum(nil))
			} else {
				result += fmt.Sprintf(" error:%s", err.Error())
			}
			break
		}
		hasher.Write(buffer[0:length])
	}
	return result
}
