package main

// this is a direct port of arc90's readbility javascript
// http://code.google.com/p/arc90labs-readability

import (
	"code.google.com/p/cascadia"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"fmt"
	//	"github.com/matrixik/goquery"
	"regexp"

//	"sort"
	"strings"
//	"os"
)


type CandidateMap map[*html.Node] *Candidate

func (candidates CandidateMap) get(n *html.Node) (*Candidate) {
	c,ok := candidates[n]
	if !ok {
		c = newCandidate(n)
		candidates[n] = c
	}
	return c
}

func (cm *CandidateMap) initializeNode(node *html.Node) {
	c := cm.get(node)

	switch node.DataAtom {
	case atom.Div: c.addScore(5,"<div>")
	case atom.Pre, atom.Td, atom.Blockquote:
		c.addScore(3,"<pre>, <td> or <blockquote>")
	case atom.Address, atom.Ol, atom.Ul, atom.Dl, atom.Dd, atom.Li, atom.Form:
		c.addScore(-3,"address, list or form")
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Th:
		c.addScore(-5,"heading")
	}

	for _,score := range getClassWeight(node) {
		c.addScore(score.Value, score.Desc)
	}
}



func DoVoodoo(root *html.Node) {
	removeScripts(root)
	// TODO: Turn all double br's into p's? Kill <style> tags? (see prepDocument())
	grabArticle(root)
}

// remove all <script> elements
func removeScripts(root *html.Node) {
	sel := cascadia.MustCompile("script")
	for _, script := range sel.MatchAll(root) {
		script.Parent.RemoveChild(script)
	}
}


var unlikelyCandidates = regexp.MustCompile(`(?i)combx|comment|community|disqus|extra|foot|header|menu|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|tweet|twitter`)
var okMaybeItsACandidate = regexp.MustCompile(`(?i)and|article|body|column|main|shadow`)

var positivePat = regexp.MustCompile(`(?i)article|body|content|entry|hentry|main|page|pagination|post|text|blog|story`)
var negativePat = regexp.MustCompile(`(?i)combx|comment|com-|contact|foot|footer|footnote|masthead|media|meta|outbrain|promo|related|scroll|shoutbox|sidebar|sponsor|shopping|tags|tool|widget`)

func grabArticle(root *html.Node) {

	var	candidates = make(CandidateMap)

	stripUnlikelyCandidates := true

	/**
	 * First, node prepping. Trash nodes that look cruddy (like ones with the class name "comment", etc), and turn divs
	 * into P tags where they have been used inappropriately (as in, where they contain no other block level elements.)
	 *
	 * Note: Assignment from index for performance. See http://www.peachpit.com/articles/article.aspx?p=31567&seqNum=5
	 * TODO: Shouldn't this be a reverse traversal?
	**/

	nodesToScore := make([]*html.Node, 0, 128)

	allNodes := cascadia.MustCompile("*")
	for _, node := range allNodes.MatchAll(root) {
		if stripUnlikelyCandidates {
			unlikelyMatchString := getAttr(node, "class") + getAttr(node, "id")
			//		fmt.Printf("? %s\n",unlikelyMatchString)
			if unlikelyCandidates.MatchString(unlikelyMatchString) == true &&
				okMaybeItsACandidate.MatchString(unlikelyMatchString) == false &&
				node.DataAtom != atom.Body {
				fmt.Printf("Removing unlikely candidate - %s\n", unlikelyMatchString)
				node.Parent.RemoveChild(node)
				continue
			}
		}

		if node.DataAtom == atom.P || node.DataAtom == atom.Td || node.DataAtom == atom.Pre {
			nodesToScore = append(nodesToScore, node)
		}
		/* XYZZY TODO: Turn all divs that don't have children block level elements into p's */
	}

	fmt.Printf("%d nodes to score\n", len(nodesToScore))

	/*
	 * Loop through all paragraphs, and assign a score to them based on how content-y they look.
	 * Then add their score to their parent node.
	 *
	 * A score is determined by things like number of commas, class names, etc. Maybe eventually link density.
	 */

	for _, node := range nodesToScore {
		parentNode := node.Parent
		var grandParentNode *html.Node = nil
		if parentNode != nil {
			grandParentNode = parentNode.Parent
		}

		innerText := getTextContent(node)
		/* If this paragraph is less than 25 characters, don't even count it. */
		if len(innerText) < 25 {
			continue
		}

		if _,exists := candidates[parentNode]; !exists {
			candidates.initializeNode(parentNode)
		}
		if grandParentNode != nil {
			if _,exists := candidates[grandParentNode]; !exists {
				candidates.initializeNode(grandParentNode)
			}
		}


		contentScore := 1

		// add points for any commas
		contentScore += strings.Count(innerText,",")

		// 1 point for every 100 bytes in this para, up to 3 points
		foo := len(innerText) / 100
		if foo > 3 {
			foo=3
		}
		contentScore += foo

        /* Add the content score to the parent. The grandparent gets half. */
		candidates.get(parentNode).addScore(contentScore,"Child content")
		if grandParentNode != nil {
			halfScore := contentScore/2
			if halfScore>0 {
				candidates.get(grandParentNode).addScore(halfScore,"Child content")
			}
		}
	}

	/**
	 * After we've calculated scores, loop through all of the possible candidate nodes we found
	 * and find the one with the highest score.
	**/
	var topCandidate *Candidate = nil;
	for _,c := range(candidates) {
		if topCandidate==nil || c.TotalScore>topCandidate.TotalScore {
			topCandidate = c
		}
	}

	for _,candidate := range(candidates) {
		candidate.dump()
	}

	topCandidate.dump()
}





/*
 * Get an elements class/id weight. Uses regular expressions to tell if this 
 * element looks good or bad.
 * returns a slice of (score,reason) pairs
**/
func getClassWeight(n *html.Node) []Score {
	//if(!readability.flagIsActive(readability.FLAG_WEIGHT_CLASSES)) {
	//    return 0;
	//}

	scores := make([]Score,0,4)

	cls := getAttr(n,"class")
	id := getAttr(n,"id")

	/* Look for a special classname */
	if negativePat.MatchString(cls) {
		scores = append(scores, Score{-25,"negative class"})
	}
	if positivePat.MatchString(cls) {
		scores = append(scores, Score{25,"indicative class"})
	}
	/* Look for a special ID */
	if negativePat.MatchString(id) {
		scores = append(scores, Score{-25,"negative id"})
	}
	if positivePat.MatchString(id) {
		scores = append(scores, Score{25,"indicative id"})
	}

	return scores
}



