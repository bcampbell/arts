package main

import (
	"net/http"
	"sync"
	"time"
)

// PoliteTripper is a http.RoundTripper implementation which imposes a
// minimum per-host delay.
//	c := &http.Client{
//		Transport: NewPoliteTripper(),
//	}
type PoliteTripper struct {
	PerHostDelay time.Duration
	lock         sync.Mutex
	nextReq      map[string]time.Time
}

func NewPoliteTripper() *PoliteTripper {
	return &PoliteTripper{PerHostDelay: 1 * time.Second, nextReq: make(map[string]time.Time)}
}

func (this *PoliteTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	now := time.Now()

	// schedule a timeslot for this req
	this.lock.Lock()
	scheduled, exists := this.nextReq[req.URL.Host]
	if !exists {
		scheduled = now
	}
	this.nextReq[req.URL.Host] = scheduled.Add(this.PerHostDelay)
	this.lock.Unlock()

	// now wait until our turn comes up
	delay := scheduled.Sub(now)
	if delay > 0 {
		//	fmt.Printf("sleep %v %s %s\n", delay, req.Method, req.URL.String())
		time.Sleep(delay)
	}
	return http.DefaultTransport.RoundTrip(req)
}

/* example usage:

func main() {

	c := &http.Client{
		Transport: NewPoliteTripper(),
	}
	var wg sync.WaitGroup

	tests := []string{
		"http://gi.dev",
		"http://gi.dev",
		"http://gi.dev",
		"http://gi.dev",
		"http://gi.dev",
		"http://gi.dev",
		"http://gi.dev",
		"http://gi.dev",
		"http://gi.dev",
		"http://jl.dev",
		"http://jl.dev",
		"http://jl.dev",
		"http://jl.dev",
		"http://jl.dev",
		"http://jl.dev",
		"http://scumways.com",
	}

	for i, u := range tests {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			fmt.Printf("%v %d: start %s\n", time.Now(), i, u)
			c.Get(u)
			fmt.Printf("%v %d: done %s\n", time.Now(), i, u)
		}(i, u)
	}

	wg.Wait()
}
*/
