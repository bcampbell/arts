package arts

import (
	"testing"
//	"os"
    "strings"
    "code.google.com/p/go.net/html"
)

type StringTest struct {
    input string
    expected string
}

var alphanumericdata = []StringTest {
    {"",""},
    {"should be unchanged","should be unchanged"},
    {"LOWERCASE","lowercase"},
    {"Hello-there-this_is_á_test \u00F1", "hellotherethisisatest n"},
}

func TestToAlphanumeric(t *testing.T) {
    for _,test := range alphanumericdata {
        got := toAlphanumeric(test.input)
        if got != test.expected {
		    t.Errorf("toAlphanumeric('%v') = '%v', want '%v'", test.input, got, test.expected)
        }
    }
}

var slugtests = []StringTest {
    {"",""},
    {"http://example.com/this-is-a-slug","this-is-a-slug"},
    {"http://example.com/strip-the-suffix.html","strip-the-suffix"},
    {"http://example.com/WIBBLE_Foo#bar","WIBBLE_Foo"},
}

func TestGetSlug(t *testing.T) {
    for _,test := range slugtests {
        got := getSlug(test.input)
        if got != test.expected {
		    t.Errorf("getSlug('%v') = '%v', want '%v'", test.input, got, test.expected)
        }
    }
}


// tests for getLinkDensity()
func TestLinkDensity(t *testing.T) {
	testData := []struct {
		htmlFragment string
		expectedDensity float64
	}{
		{`<p>Hello.</p><p>No links here</p>`, 0},
		{`<p><a href="#">It's all linkage!</a></p>`, 1},
		{`<div><a href="#">Half is link</a> half is not</div>`, 0.5},
		{`<div><p>Quarter of this is links. <a href="#">here!</a> + <a href="#">here!</a>.</div>`, 0.25},
	}

	for _,dat := range testData {
		node, _ := html.Parse(strings.NewReader(dat.htmlFragment))
		got := getLinkDensity(node)
		if got != dat.expectedDensity {
			t.Errorf("getLinkDensity('%s') = %v (expected %v)", dat.htmlFragment, got, dat.expectedDensity)
		}
	}
}

