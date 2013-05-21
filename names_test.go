package arts

import (
	"fmt"
	"testing"
)

// test for rateName()
func TestRateName(t *testing.T) {

	names := []string{
		`Jacobus Henricus van 't Hoff`,
		`Bob Roberts`,
		`Homer J. Simpson`,
		`Homer J Simpson`,
		`Alberto Santos-Dumont`,
	}

	notNames := []string{
		"about",
	}

	for _, n := range names {
		score := rateName(n)
		fmt.Printf("%s: %f\n", n, score)
		if score <= 0 {
			t.Errorf(`rateName("%s") returned %f (expected>0)")`, n, score)
		}
	}
	for _, n := range notNames {
		score := rateName(n)
		fmt.Printf("%s: %f\n", n, score)
		if score > 0 {
			t.Errorf(`rateName("%s") returned %f (expected<=0)")`, n, score)
		}
	}

	//      t.Errorf(`bad canonical (got "%s" expected "%s")`, canonical, expectedCanonical)
}
