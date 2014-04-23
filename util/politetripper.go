package util

import (
	//	"fmt"
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
	prevTime     map[string]time.Time
}

func NewPoliteTripper() *PoliteTripper {
	return &PoliteTripper{PerHostDelay: 1 * time.Second, prevTime: make(map[string]time.Time)}
}

func (this *PoliteTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for {
		this.lock.Lock()
		prev, _ := this.prevTime[req.URL.Host]
		elapsed := time.Since(prev)

		if elapsed >= this.PerHostDelay {
			// OK - go!
			this.prevTime[req.URL.Host] = time.Now()
			this.lock.Unlock()
			//fmt.Printf("%s: GO %v\n", req.URL.Host, time.Now())
			return http.DefaultTransport.RoundTrip(req)

		}
		this.lock.Unlock()

		// sleep until we think the expected time has passed
		// (but some other request might get in first, hence inifinte for
		// loop to repeating the whole thing from scratch...
		delay := this.PerHostDelay - elapsed
		//		fmt.Printf("%s: sleep %v\n", req.URL.Host, delay)
		time.Sleep(delay)
	}
}

/* example usage:

func main() {

	c := &http.Client{
		Transport: NewPoliteTripper(),
	}

	tests := []string{
		"http://example.com",
		"http://example.com",
		"http://example.com",
		"http://example.com",
		"http://example.com",
	}

	for i, u := range tests {
        fmt.Printf("%v %d: start %s\n", time.Now(), i, u)
        c.Get(u)
        fmt.Printf("%v %d: done %s\n", time.Now(), i, u)
	}

}
*/
