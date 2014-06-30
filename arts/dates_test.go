package arts

import (
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
	}
	// go for it.
	for _, dat := range testData {

		dt := dateFromURL(dat.url)

		if dt.String() != dat.expect {
			t.Errorf(`bad date from url (got "%s" expected "%s")`, dt.String(), dat.expect)
		}

	}
}
