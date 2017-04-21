package arts

import (
	"net/url"
	"testing"
)

// TestDateFromURL tests the dateFromURL() function
func TestDateFromURL(t *testing.T) {

	testData := []struct {
		url    string
		expect string
	}{
		{"http://www.example.com/posts/2014/04/17/moon-made-of-cheese",
			"2014-04-17"},
        { "http://www.itv.com/news/2017-04-21/may-under-pressure-to-outline-pension-plans-after-making-foreign-aid-pledge/",
        "2017-04-21"},
	}
	// go for it.
	for _, dat := range testData {

		u, err := url.Parse(dat.url)
		if err != nil {
			t.Errorf(`bad url "%s"`, dat.url)
		}
		dt := dateFromURL(u)

		if dt.String() != dat.expect {
			t.Errorf(`bad date from url (got "%s" expected "%s")`, dt.String(), dat.expect)
		}

	}
}
