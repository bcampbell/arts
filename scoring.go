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
	TotalScore int
    Scores []Score
}

func (c *Candidate) addScore(value int,desc string) {
    c.Scores = append(c.Scores,Score{value,desc})
    c.TotalScore += value
}

// print out a candidate and the scores it received for debugging
func (c *Candidate) dump() {
    fmt.Printf("%d %s '%s'\n", c.TotalScore, c.Node.DataAtom.String(), c.Txt)
    for _,s := range(c.Scores) {
        fmt.Printf("  %d %s\n", s.Value, s.Desc)
    }
}

// implements a sortable set of Candidates
type Candidates []Candidate

func (s Candidates) Len() int           { return len(s) }
func (s Candidates) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Candidates) Less(i, j int) bool { return s[i].TotalScore < s[j].TotalScore }




// wrapper for reversing any sortable
type Reverse struct {
	sort.Interface
}

func (r Reverse) Less(i, j int) bool {
	return r.Interface.Less(j, i)
}


