package arts

import (
	"golang.org/x/net/html"
	"net/url"
	"sort"
	"strings"
	"testing"
)

// TestGrabUrls tests the grabURLs() function
func TestGrabURLs(t *testing.T) {

	testData := []struct {
		rawHTML   string
		srcURL    string
		canonical string
		urls      []string
	}{
		// test 1: no alternative urls on page
		{`<html><head></head><body></body></html>`,
			"http://example.com/fook",
			"",
			[]string{"http://example.com/fook"},
		},
		// test 2: check canonical extraction, url normalisation
		{
			`<!DOCTYPE html>
<html>
 <head>
  <meta property="og:url" content="http://example.com/fook?fb=1" />
  <link rel="canonical" href="HTTP://Example.Com/fook" />
  <link rel="shortlink" href="http://examp.le/?p=1" />
 </head>
 <body>
 </body>
</html>
`,
			"http://example.com/fook",
			"http://example.com/fook",
			[]string{"http://example.com/fook", "http://example.com/fook?fb=1", "http://examp.le/?p=1"},
		},
		// reject obvious root urls
		{`<html><head><link rel="canonical" href="http://example.com" /></head><body></body></html>`,
			"http://example.com/fook",
			"",
			[]string{"http://example.com/fook"},
		},
	}

	// go for it.
	for _, expected := range testData {

		srcUrl, err := url.Parse(expected.srcURL)
		if err != nil {
			panic(err)
		}

		sort.Strings(expected.urls)

		root, err := html.Parse(strings.NewReader(expected.rawHTML))
		if err != nil {
			panic(err)
		}

		canonical, all := grabURLs(root, srcUrl)

		if canonical != expected.canonical {
			t.Errorf(`bad canonical (got "%s" expected "%s")`, canonical, expected.canonical)
		}

		bad := false
		sort.Strings(all)
		if len(all) != len(expected.urls) {
			bad = true
		} else {
			for i, alt := range all {
				if alt != expected.urls[i] {
					bad = true
				}
			}
		}

		if bad {
			t.Errorf(`bad url list (got "%v" expected "%v")`, all, expected.urls)
		}
	}
}
