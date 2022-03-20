package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

// RoundTripFunc
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
		Timeout:   time.Duration(1) * time.Second,
	}
}

// TestCaseForProcessFunc is struct for TestProccess function
type TestCaseForProcessFunc struct {
	InputUrls          map[string]string
	InputParallelCount int
}

// TestProccess checks correctness of proccess function
func TestProcess(t *testing.T) {
	urlsResp := map[string]string{
		"http://one": "one",
		"http://two": "two",
	}
	// 1 url 1 parallel count
	// 2 url 1 parallel count
	// 2 url 2 parallel count
	tests := []TestCaseForProcessFunc{
		TestCaseForProcessFunc{
			InputUrls: map[string]string{
				"http://one": urlsResp["http://one"],
			},
			InputParallelCount: 1,
		},
		TestCaseForProcessFunc{
			InputUrls: map[string]string{
				"http://one": urlsResp["http://one"],
				"http://two": urlsResp["http://two"],
			},
			InputParallelCount: 1,
		},
		TestCaseForProcessFunc{
			InputUrls: map[string]string{
				"http://one": urlsResp["http://one"],
				"http://two": urlsResp["http://two"],
			},
			InputParallelCount: 2,
		},
	}
	fn := func(req *http.Request) *http.Response {
		respBody, ok := urlsResp[req.URL.String()]
		if !ok {
			fmt.Printf("ERROR: TestProcess try to test url without response. Put Values in urlsResp. Url: %s\n", req.URL.String())
		}
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(respBody)),
			Header:     make(http.Header),
		}
	}
	client := NewTestClient(fn)
	for _, test := range tests {
		urls := []string{}
		var expectedOutput string
		for url, body := range test.InputUrls {
			urls = append(urls, url)
			if expectedOutput != "" {
				expectedOutput += " "
			}
			expectedOutput += fmt.Sprintf("%s %x", url, md5.Sum([]byte(body)))
		}
		output := Process(client, 1, urls)
		if output != expectedOutput {
			t.Errorf("Failed proccess, expected: %s, got: %s. Input: %v. ParallelCount:%d", expectedOutput, output, test.InputUrls, test.InputParallelCount)
		}
	}
}

// TestRequestSuccess checks correctness of reading body and calculating md5 hash sum
func TestRequestSuccess(t *testing.T) {
	tests := map[string]string{
		"http://smallBody.ru": "testBody",
		"http://largeBody.ru": strings.Repeat("x", 4096),
	}

	fn := func(req *http.Request) *http.Response {
		respBody, ok := tests[req.URL.String()]
		if !ok {
			fmt.Printf("ERROR: TestRequestSuccess try to test url without response. Put Values in tests map. Url: %s\n", req.URL.String())
		}
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(respBody)),
			Header:     make(http.Header),
		}
	}
	client := NewTestClient(fn)

	for url, body := range tests {
		expectedOutput := fmt.Sprintf("%s %x", url, md5.Sum([]byte(body)))
		output := Request(client, url)
		if output != expectedOutput {
			t.Errorf("Failed Request, expected: %s, got: %s", expectedOutput, output)
		}
	}
}
func TestRequestFailed(t *testing.T) {
	fn := func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 408,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`timeout`)),
			Header:     make(http.Header),
		}
	}
	client := NewTestClient(fn)

	url := "http://timeout.ru"
	expectedOutput := fmt.Sprintf("%s error:408", url)
	output := Request(client, url)
	if output != expectedOutput {
		t.Errorf("Failed Request, expected: %s, got: %s", expectedOutput, output)
	}
}

// TestCaseForGetParallelCount  is struct for TestGetParallelCount function
type TestCaseForGetParallelCount struct {
	InputParallelCount  int
	InputUrlsCount      int
	OutputParallelCount int
}

// TestGetParallelCount checks value of parallel requests returning by getParallelCount function
func TestGetParallelCount(t *testing.T) {
	tests := []TestCaseForGetParallelCount{
		{-1, 10, 1},
		{-1, 1, 1},
		{11, 10, 10},
		{11, 1, 1},
	}
	for _, test := range tests {
		output := getParallelCount(test.InputParallelCount, test.InputUrlsCount)
		if output != test.OutputParallelCount {
			t.Errorf("Failed getParallelCount, expected: %d, got: %d. Input:[parallel:%d urlsCount: %d]", test.OutputParallelCount, output, test.InputParallelCount, test.InputUrlsCount)
		}
	}
}

// TestCaseForGetURLS is struct for TestGetURLS function
type TestCaseForGetURLS struct {
	Input  []string
	Output []string
	Error  error
}

func TestGetURLS(t *testing.T) {
	tests := []TestCaseForGetURLS{
		TestCaseForGetURLS{
			Input:  []string{"twitter.com"},
			Output: []string{"http://twitter.com"},
		},
		TestCaseForGetURLS{
			Input:  []string{"twitter.com", "http://twitter.com"},
			Output: []string{"http://twitter.com"},
		},
		TestCaseForGetURLS{
			Input:  []string{"twitter.com", "http://ya.ru"},
			Output: []string{"http://twitter.com", "http://ya.ru"},
		},
		TestCaseForGetURLS{
			Input:  []string{},
			Output: []string{},
			Error:  NoUrlsError,
		},
	}
	for _, test := range tests {
		output, err := getURLS(test.Input)
		if test.Error == nil {
			if ok := reflect.DeepEqual(output, test.Output); !ok {
				t.Errorf("Failed getURLS, expected: %v, got: %v. Input:%v", test.Output, output, test.Input)
			}
		} else {
			if err != test.Error {
				t.Errorf("Failed getURLS error case, expected: \"%v\", got: \"%v\". Input:%v", test.Error, err, test.Input)
			}
		}
	}
}
