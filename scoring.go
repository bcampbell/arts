package main
import (
	"fmt"
	"sort"
	"code.google.com/p/go.net/html"
//	"strings"
)


// some points, along with the reason the points were assigned
type Score struct {
    Value int
    Desc string
}

// for rating a node/text snippet
type Candidate struct {
	Node  *html.Node
	Txt   string
	TotalScore float64
    Log []string
}

func newCandidate(n *html.Node, txt string) *Candidate {
	return &Candidate{n, txt, 0, make([]string,0,4)}
}

func (c *Candidate) addScore(value float64,desc string) {
	c.Log = append(c.Log,fmt.Sprintf("%+.3g %s",value, desc))
    c.TotalScore += value
}

func (c *Candidate) scaleScore(scaleFactor float64,desc string) {
	c.Log = append(c.Log,fmt.Sprintf("*%.3g %s",scaleFactor, desc))
    c.TotalScore *= scaleFactor
}

// print out a candidate and the scores it received for debugging
func (c *Candidate) dump() {
    fmt.Printf("%.3g %s '%s'\n", c.TotalScore, describeNode(c.Node), c.Txt)
    for _,s := range(c.Log) {
        fmt.Printf("  %s\n", s)
    }
}

// implements a sortable set of Candidates
type CandidateList []*Candidate

func (s CandidateList) Len() int           { return len(s) }
func (s CandidateList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s CandidateList) Less(i, j int) bool { return s[i].TotalScore < s[j].TotalScore }




// wrapper for reversing any sortable
type Reverse struct {
	sort.Interface
}

func (r Reverse) Less(i, j int) bool {
	return r.Interface.Less(j, i)
}


