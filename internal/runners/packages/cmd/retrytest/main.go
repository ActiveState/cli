package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jarcoal/httpmock"
)

func main() {
	var (
		url      = "http://test.thisout.tld"
		testPath = url + "/tester"
	)
	defer fmt.Println("main - here")

	t := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	cl := &http.Client{
		Timeout:   time.Second * 10,
		Transport: t,
	}

	httpmock.ActivateNonDefault(cl)
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", testPath,
		httpmock.NewStringResponder(200, `[{"id": 1, "name": "My Great Article"}]`),
	)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer fmt.Println("go func - here")

		resp, err := cl.Get(testPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer resp.Body.Close()
	}()

	rcl := retryablehttp.NewClient()
	rcl.HTTPClient = cl

	resp, err := rcl.Get(testPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	defer resp.Body.Close()

	wg.Wait()
}
