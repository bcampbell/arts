package util

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func ExamplePoliteClient() {

	// create a client which never hits hosts at more than 1 req/sec
	c := &http.Client{
		Transport: NewPoliteTripper(),
	}

	tests := []string{
		// five reqs to example.com => at least 4 delays between them
		"http://example.com",
		"http://example.com",
		"http://example.com",
		"http://example.com",
		"http://example.com",
		"http://example.com",
		"http://example.com",
	}

	startTime := time.Now()
	for _, u := range tests {
		c.Get(u)
	}
	elapsed := time.Now().Sub(startTime)
	if elapsed >= 4*time.Second {
		fmt.Println("As expected, took at least 4 seconds")
	} else {
		fmt.Println("uhoh... took <4 seconds!")
	}
	// Output:
	// As expected, took at least 4 seconds
}

// Concurrent example
func ExamplePoliteClient2() {

	// create a client which never hits hosts at more than 1 req/sec
	c := &http.Client{
		Transport: NewPoliteTripper(),
	}
	var wg sync.WaitGroup

	// a bunch of requests to issue all at once
	tests := []string{
		// five reqs to example.com => at least 4 delays between them
		"http://example.com",
		"http://example.com",
		"http://example.com",
		"http://example.com",
		"http://example.com",
	}

	startTime := time.Now()
	for _, u := range tests {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			//			fmt.Printf("start %s\n", u)
			c.Get(u)
			//			fmt.Printf("done %s\n", u)
		}(u)
	}

	wg.Wait()
	elapsed := time.Now().Sub(startTime)
	if elapsed >= 4*time.Second {
		fmt.Println("As expected, took at least 4 seconds")
	} else {
		fmt.Println("uhoh... took <4 seconds!")
	}
	// Output:
	// As expected, took at least 4 seconds
}
