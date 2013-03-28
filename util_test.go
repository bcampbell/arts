package main


import "testing"

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
