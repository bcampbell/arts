package arts

import (
	"code.google.com/p/go.net/html"
	"sort"
	"strings"
	"testing"
)

// TestGrabUrls tests the grabUrls() function
func TestGrabUrls(t *testing.T) {

	testHtml := `<!DOCTYPE html>
<html>
 <head>
  <meta property="og:url" content="http://example.com/fook?fb=1" />
  <link rel="canonical" href="http://example.com/fook" />
  <link rel="shortlink" href="http://examp.le/?p=1" />
 </head>
 <body>
 </body>
</html>
`
	expectedCanonical := "http://example.com/fook"
	expectedAlternates := []string{"http://example.com/fook?fb=1", "http://examp.le/?p=1"}
	sort.Strings(expectedAlternates)

	root, err := html.Parse(strings.NewReader(testHtml))
	if err != nil {
		panic(err)
	}

	canonical, alternates := grabUrls(root)

	if canonical != expectedCanonical {
		t.Errorf(`bad canonical (got "%s" expected "%s")`, canonical, expectedCanonical)
	}

	bad := false
	sort.Strings(alternates)
	if len(alternates) != len(expectedAlternates) {
		bad = true
	} else {
		for i, alt := range alternates {
			if alt != expectedAlternates[i] {
				bad = true
			}
		}
	}

	if bad {
		t.Errorf(`bad alternate url list (got "%v" expected "%v")`, alternates, expectedAlternates)
	}
}
