package byline

import (
	"fmt"
	//	"testing"
	"strings"
)

func ExampleParse() {
	bylines := []string{
		"By Matthew Garrahan in Los Angeles and Tim Bradshaw in San Francisco",
		"Fred Blogs, in Washington and Bubba jo-bob Brain, chief cheese editor in Mouseland",
		"Sports Reporter",
		"Fred Smith",
		"By SARA KARL. Special to amNewYork April 24, 2014",
		"By Lucy Hyslop, Special to The Sun",
		"Daniel Wittenberg for Metro.co.uk",
	}

	for _, byl := range bylines {
		authors := Parse(byl)
		names := make([]string, len(authors))
		for i, a := range authors {
			names[i] = a.Name
		}
		fmt.Println(strings.Join(names, "|"))
	}

	// Output:
	// Matthew Garrahan|Tim Bradshaw
	// Fred Blogs|Bubba jo-bob Brain
	//
	// Fred Smith
	// SARA KARL
	// Lucy Hyslop
}
