package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/retryfn"
)

func main() {
	c := make(chan struct{})
	timer := time.AfterFunc(5*time.Second, func() {
		close(c)
	})

	// Serve 256 bytes every second.
	req, err := http.NewRequest("GET", "http://httpbin.org/range/2048?duration=8&chunk_size=256", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Cancel = c

	fn := func() error {
		log.Println("Sending request...")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		log.Println("Reading body...")
		timer.Reset(2 * time.Second)
		timer.Reset(50 * time.Millisecond) // try instead
		_, err = io.CopyN(ioutil.Discard, io.TeeReader(resp.Body, os.Stdout), 256)
		if err != nil && err != io.EOF {
			return err
		}

		return nil
	}

	retryFn := retryfn.New(3, fn)
	log.Println(retryFn.Run())
}
