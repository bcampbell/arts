package arts

// scoring.go contains helpers for rating nodes.

import (
	"fmt"
	"code.google.com/p/go.net/html"
//	"strings"
)



// Candidate is for rating a node/text snippet.
// it keeps a little log of the accumulating scoring operations to aid
// debugging.
// (at the end of processing, it's very useful to be able to see what
// happened to a particular candidate along the way. Saves us the shotgun
// approach of logging everything as it happens, then trying to read back
// through it)
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

// dump prints out a candidate and the scores it received for debugging
func (c *Candidate) dump() {
    fmt.Printf("%.3g %s '%s'\n", c.TotalScore, describeNode(c.Node), c.Txt)
    for _,s := range(c.Log) {
        fmt.Printf("  %s\n", s)
    }
}

// Candidate implements a sortable set of Candidates
type CandidateList []*Candidate

func (s CandidateList) Len() int           { return len(s) }
func (s CandidateList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s CandidateList) Less(i, j int) bool { return s[i].TotalScore < s[j].TotalScore }

// CandidateMap stores candidates for quick lookup by node
type CandidateMap map[*html.Node]*Candidate

// get returns an existing candidiate struct or create a blank new one
func (candidates CandidateMap) get(n *html.Node) *Candidate {
	c, ok := candidates[n]
	if !ok {
		c = newCandidate(n, "")
		candidates[n] = c
	}
	return c
}




