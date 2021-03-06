package arts

import (
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	//	"fmt"
	//	"os"
	"math"
	"strings"
	"testing"
)

type StringTest struct {
	input    string
	expected string
}

var alphanumericdata = []StringTest{
	{"", ""},
	{"should be unchanged", "should be unchanged"},
	{"LOWERCASE", "lowercase"},
	{"Hello-there-this_is_á_test \u00F1", "hellotherethisisatest n"},
}

func TestToAlphanumeric(t *testing.T) {
	for _, test := range alphanumericdata {
		got := toAlphanumeric(test.input)
		if got != test.expected {
			t.Errorf("toAlphanumeric('%v') = '%v', want '%v'", test.input, got, test.expected)
		}
	}
}

var slugtests = []StringTest{
	{"", ""},
	{"http://example.com/this-is-a-slug", "this-is-a-slug"},
	{"http://example.com/strip-the-suffix.html", "strip-the-suffix"},
	{"http://example.com/WIBBLE_Foo#bar", "WIBBLE_Foo"},
	{"http://www.stuff.co.nz/southland-times/business/8822601/Mataura-briquetting-plant-on-market", "Mataura-briquetting-plant-on-market"},
}

func TestGetSlug(t *testing.T) {
	for _, test := range slugtests {
		got := getSlug(test.input)
		if got != test.expected {
			t.Errorf("getSlug('%v') = '%v', want '%v'", test.input, got, test.expected)
		}
	}
}

// tests for getLinkDensity()
func TestLinkDensity(t *testing.T) {
	testData := []struct {
		htmlFragment    string
		expectedDensity float64
	}{
		{`<p>Hello.</p><p>No links here</p>`, 0},
		{`<p><a href="#">It's all linkage!</a></p>`, 1},
		{`<div><a href="#">Half is link</a> half is not</div>`, 0.5},
		{`<div><p>Quarter of this is links. <a href="#">here!</a> + <a href="#">here!</a>.</div>`, 0.25},
	}

	for _, dat := range testData {
		node, _ := html.Parse(strings.NewReader(dat.htmlFragment))
		got := getLinkDensity(node)
		if got != dat.expectedDensity {
			t.Errorf("getLinkDensity('%s') = %v (expected %v)", dat.htmlFragment, got, dat.expectedDensity)
		}
	}
}

// test for prevNode()
/*
func TestPrevNode(t *testing.T) {
	htmlFragment := `<html>
    <head>
    <title>PageTitle</title>
</head>
<body>
<p>paragraph one <span>one</span></p>
<p>paragraph <a id="two">two</a>.</p>
</body>
</html>`
	root, _ := html.Parse(strings.NewReader(htmlFragment))
	sel := cascadia.MustCompile("#two")
	n := sel.MatchAll(root)[0]
	fmt.Printf("%s\n------\n", htmlFragment)
	for ; n != nil; n = prevNode(n) {
		fmt.Printf("%s\n", describeNode(n))
	}
}
*/

func TestWordCount(t *testing.T) {
	testData := []struct {
		s        string
		expected int
	}{
		{``, 0},
		{`simple`, 1},
		{"some\nlines\nof\ntext.\n", 4},
		{`  some surrounding space   `, 3},
	}
	for _, dat := range testData {
		got := wordCount(dat.s)
		if got != dat.expected {
			t.Errorf("wordCount('%s') = %v (expected %v)", dat.s, got, dat.expected)
		}
	}

}

func TestJaccardWordCompare(t *testing.T) {
	testData := []struct {
		needle   string
		haystack string
		expected float64
	}{
		{"full match", "full match", 1},
		{"order ignored", "ignored order", 1},
		{"case SENSITIVE", "CASE sensitive", 0},
		{"no match at all", "fishy fishy fishy", 0},
		{"one two", "one", 0.5},
		{"half of a match", "half of wibble pibble", 0.333333},
		{"sub set of words", "should match a sub set of words even if surrounded", 0.4},
		{"most words matching but not", "most words matching but not all", 0.83333},
		{"Торт настолько вкусный и влажный.", "Торт настолько вкусный и влажный.", 1},
		{"", "", 1},
	}
	for _, dat := range testData {
		got := jaccardWordCompare(dat.haystack, dat.needle)
		if math.IsNaN(got) {
			t.Errorf("jaccardWordCompare('%s','%s') = %v ", dat.haystack, dat.needle, got)
		}
		if math.Abs(dat.expected-got) > 0.001 {
			t.Errorf("jaccardWordCompare('%s','%s') = %v (expected %v)", dat.haystack, dat.needle, got, dat.expected)
		}
	}
}

var walkHTML string = `<html>
  <head></head>
  <body>
    <div id="a">
      <div id="b">
        <div id="c">
        </div>
        <div id="d">
        </div>
      </div>
    </div>
    <div id="e">
    </div>
  </body>
</html>
`

func parseDoc(src string) *html.Node {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		panic(err)
	}
	return doc
}

func TestNextElement(t *testing.T) {

	cases := []struct {
		start    string
		expected string
	}{
		{"html", "head"},
		{"head", "body"},
		{"#c", "#d"},
		{"#d", "#e"},
	}

	doc := parseDoc(walkHTML)

	for _, dat := range cases {
		e := cascadia.MustCompile(dat.start).MatchFirst(doc)
		expect := cascadia.MustCompile(dat.expected).MatchFirst(doc)

		got := nextElement(e)

		//fmt.Printf("%s => %s\n", describeNode(e), describeNode(got))
		if got != expect {
			t.Errorf("nextElement('%s') got %s (expected %s)", dat.start, describeNode(got), dat.expected)
		}
	}
}

func TestInterveningElements(t *testing.T) {

	cases := []struct {
		e1Sel        string
		e2Sel        string
		expectedSels []string
	}{
		{"#a", "#e", []string{"#b", "#c", "#d"}},
		{"html", "body", []string{"head"}},
	}

	doc := parseDoc(walkHTML)

	for _, dat := range cases {
		e1 := cascadia.MustCompile(dat.e1Sel).MatchFirst(doc)
		e2 := cascadia.MustCompile(dat.e2Sel).MatchFirst(doc)
		expected := []*html.Node{}
		for _, sel := range dat.expectedSels {
			expected = append(expected, cascadia.MustCompile(sel).MatchFirst(doc))
		}

		got, err := interveningElements(e1, e2)
		if err != nil {
			t.Errorf("interveningElements(%s,%s) failed: %s", dat.e1Sel, dat.e2Sel, err)
			break
		}

		if len(got) != len(expected) {
			t.Errorf("interveningElements(%s,%s) got: %v  expected: %v", dat.e1Sel, dat.e2Sel, got, expected)
			break
		}
		// TODO: elementwise compare
	}
}
