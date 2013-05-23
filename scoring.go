package arts

// scoring.go contains helpers for rating nodes.

import (
	"code.google.com/p/go.net/html"
	"fmt"
	"io"
	"strconv"

//	"strings"
)

type candidate interface {
	addPoints(value float64, desc string)
	scalePoints(scaleFactor float64, desc string)
	total() float64
	dump(out io.Writer)
	txt() string
	node() *html.Node
}

// standardCandidate implements a candidate to hold a node ptr and it's text
// content.
// it keeps a little log of the accumulating scoring operations to aid
// debugging.
// (at the end of processing, it's very useful to be able to see what
// happened to a particular candidate along the way. Saves us the shotgun
// approach of logging everything as it happens then trying to read back
// through it)
type standardCandidate struct {
	n      *html.Node
	t      string
	points float64
	scale  float64
	log    []string
}

func newStandardCandidate(n *html.Node, txt string) candidate {
	return &standardCandidate{n, txt, 0, 1, make([]string, 0, 4)}
}

func (c *standardCandidate) addPoints(value float64, desc string) {
	c.log = append(c.log, fmt.Sprintf("%+.3g %s", value, desc))
	c.points += value
}

func (c *standardCandidate) scalePoints(scaleFactor float64, desc string) {
	c.log = append(c.log, fmt.Sprintf("*%.3g %s", scaleFactor, desc))
	c.scale *= scaleFactor
}

func (c *standardCandidate) total() float64 {
	return c.points * c.scale
}

// dump prints out a candidate and the scores it received for debugging
func (c *standardCandidate) dump(out io.Writer) {
	fmt.Fprintf(out, "%.3g %s %s\n", c.total(), describeNode(c.node()), strconv.Quote(c.txt()))
	for _, s := range c.log {
		fmt.Fprintf(out, "  %s\n", s)
	}
}

func (c *standardCandidate) txt() string {
	return c.t
}
func (c *standardCandidate) node() *html.Node {
	return c.n
}

// Candidate implements a sortable set of Candidates
type candidateList []candidate

func (s candidateList) Len() int           { return len(s) }
func (s candidateList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s candidateList) Less(i, j int) bool { return s[i].total() < s[j].total() }
