package discover

import (
	"errors"
	"net/url"
)

// thin map wrapper for some set operations
type LinkSet map[url.URL]bool

// remove and return a single item from the set
func (s *LinkSet) Pop() url.URL {
	for u, _ := range *s {
		delete(*s, u)
		return u
	}
	// panic!
	panic(errors.New("tried to Pop() on empty LinkSet"))
}

func (s *LinkSet) Add(link url.URL) {
	(*s)[link] = true
}

func (s *LinkSet) Remove(link url.URL) {
	delete(*s, link)
}

// merge the contents of other into this set
func (s *LinkSet) Merge(other LinkSet) {
	for link, _ := range other {
		(*s)[link] = true
	}
}
