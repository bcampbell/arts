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
	"math"
	//	"sort"
	"strings"

	"os"
)

type CandidateMap map[*html.Node]*Candidate

func (candidates CandidateMap) get(n *html.Node) *Candidate {
	c, ok := candidates[n]
	if !ok {
		c = newCandidate(n, "")
		candidates[n] = c
	}
	return c
}

func (cm *CandidateMap) initializeNode(node *html.Node) {
	c := cm.get(node)

	switch node.DataAtom {
	case atom.Div:
		c.addScore(5, "<div>")
	case atom.Pre, atom.Td, atom.Blockquote:
		c.addScore(3, "<pre>, <td> or <blockquote>")
	case atom.Address, atom.Ol, atom.Ul, atom.Dl, atom.Dd, atom.Li, atom.Form:
		c.addScore(-3, "address, list or form")
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Th:
		c.addScore(-5, "heading")
	}

	if score := getClassWeight(node); score != 0 {
		c.addScore(score, "class/id score")
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

	var candidates = make(CandidateMap)

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
				fmt.Printf("Removing unlikely candidate - %s\n", describeNode(node))
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

	// TODO: make sure order is right, so proper scores propogate
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

		if _, exists := candidates[parentNode]; !exists {
			candidates.initializeNode(parentNode)
		}
		if grandParentNode != nil {
			if _, exists := candidates[grandParentNode]; !exists {
				candidates.initializeNode(grandParentNode)
			}
		}

		contentScore := 1.0

		// add points for any commas
		contentScore += float64(strings.Count(innerText, ","))

		// 1 point for every 100 bytes in this para, up to 3 points
		foo := float64(len(innerText)) / 100
		if foo > 3 {
			foo = 3
		}
		contentScore += foo

		/* Add the content score to the parent. The grandparent gets half. */
		candidates.get(parentNode).addScore(contentScore, "Child content")
		if grandParentNode != nil {
			halfScore := contentScore / 2
			if halfScore > 0 {
				candidates.get(grandParentNode).addScore(halfScore, "Child content")
			}
		}
	}

	/**
	 * Scale the final candidates score based on link density. Good content should have a
	 * relatively small link density (5% or less) and be mostly unaffected by this operation.
	 **/
	for _, c := range candidates {
		c.scaleScore((1 - getLinkDensity(c.Node)), "link density")
	}

	/**
	 * After we've calculated scores, loop through all of the possible candidate nodes we found
	 * and find the one with the highest score.
	**/
	var topCandidate *Candidate = nil
	for _, c := range candidates {
		if topCandidate == nil || c.TotalScore > topCandidate.TotalScore {
			topCandidate = c
		}
	}

	/**
	 * Now that we have the top candidate, look through its siblings for content that might also be related.
	 * Things like preambles, content split by ads that we removed, etc.
	**/

	siblingScoreThreshold := topCandidate.TotalScore * 0.2
	if siblingScoreThreshold < 10 {
		siblingScoreThreshold = 10
	}

	contentNodes := make([]*html.Node, 0, 64)

	for siblingNode := topCandidate.Node.Parent.FirstChild; siblingNode != nil; siblingNode = siblingNode.NextSibling {
		useIt := false
		if siblingNode == topCandidate.Node {
			useIt = true
		} else {

			contentBonus := 0.0
			/* Give a bonus if sibling nodes and top candidates have the exact same classname */
			topClass := getAttr(topCandidate.Node, "class")
			if getAttr(siblingNode, "class") == topClass && topClass != "" {
				contentBonus += topCandidate.TotalScore * 0.2
			}
			if sc, ok := candidates[siblingNode]; ok == true && sc.TotalScore+contentBonus >= siblingScoreThreshold {
				useIt = true
			}

			if siblingNode.DataAtom == atom.P {
				linkDensity := getLinkDensity(siblingNode)
				nodeContent := getTextContent(siblingNode)
				nodeLength := len(nodeContent)

				if nodeLength >= 80 && linkDensity < 0.25 {
					useIt = true
				} else if nodeLength < 80 && linkDensity == 0 && regexp.MustCompile(`\.( |$)`).MatchString(nodeContent) {
					useIt = true
				}
			}
		}

		if useIt {
			contentNodes = append(contentNodes, siblingNode)
		}

	}

	//	for _, candidate := range candidates {
	//		candidate.dump()
	//	}
	//	fmt.Printf("best:\n")
	//	topCandidate.dump()



	// go through and clean out any cruft
	for _,node := range contentNodes {
        cleanConditionally(node, "form",candidates)
        cleanConditionally(node, "table",candidates)
        cleanConditionally(node, "ul",candidates)
        cleanConditionally(node, "div",candidates)
	}

	fmt.Printf("picked %d nodes:\n", len(contentNodes))
	for _, n := range contentNodes {
		fmt.Printf("%s:\n", describeNode(n))
		html.Render(os.Stdout,n)
	}

}

/*
 * Get an elements class/id weight. Uses regular expressions to tell if this
 * element looks good or bad.
**/
func getClassWeight(n *html.Node) float64 {
	//if(!readability.flagIsActive(readability.FLAG_WEIGHT_CLASSES)) {
	//    return 0;
	//}

	score := 0.0

	cls := getAttr(n, "class")
	id := getAttr(n, "id")

	/* Look for a special classname */
	if negativePat.MatchString(cls) {
		score -= 25
	}
	if positivePat.MatchString(cls) {
		score += 25
	}
	/* Look for a special ID */
	if negativePat.MatchString(id) {
		score -= 25
	}
	if positivePat.MatchString(id) {
		score += 25
	}

	return score
}

/*
 * Prepare the article nodes for display. Clean out any inline styles,
 * iframes, forms, strip extraneous <p> tags, etc.
 *
 */
func prepArticle(articleContent []*html.Node) {
}

/**
 * Clean an element of all tags of type "tag" if they look fishy.
 * "Fishy" is an algorithm based on content length, classnames, link density, number of images & embeds, etc.
 **/
func cleanConditionally(e *html.Node, tagSel string, candidates CandidateMap) {
	sel := cascadia.MustCompile(tagSel)
	toRemove := false
	doomed := make([]*html.Node,0,32)
	for _, node := range sel.MatchAll(e) {
		weight := getClassWeight(node)
		var contentScore float64 = 0.0
		if c, ok := candidates[node]; ok {
			contentScore = c.TotalScore
		}

		if weight+contentScore < 0 {
			toRemove = true
		} else {
			textContent := getTextContent(node)
			if strings.Count(textContent, ",") < 10 {
				/*
				 * If there are not very many commas, and the number of
				 * non-paragraph elements is more than paragraphs or other ominous signs, remove the element.
				 */
				p := len(cascadia.MustCompile("p").MatchAll(node))
				img := len(cascadia.MustCompile("img").MatchAll(node))
				li := len(cascadia.MustCompile("li").MatchAll(node))
				input := len(cascadia.MustCompile("input").MatchAll(node))
				// XYZZY TODO: exclude videos?
				embedCount := len(cascadia.MustCompile("embed").MatchAll(node))
				linkDensity := getLinkDensity(node)

				if  img > p  {
                    toRemove = true
                } else if li > p && node.DataAtom != atom.Ul && node.DataAtom != atom.Ol {
                    toRemove = true
                } else if input > int(math.Floor(float64(p)/3.0)) {
                    toRemove = true
                } else if len(textContent) < 25 && (img == 0 || img > 2) {
                    toRemove = true
                } else if weight < 25 && linkDensity > 0.2 {
                    toRemove = true
                } else if weight >= 25 && linkDensity > 0.5 {
                    toRemove = true
                } else if (embedCount == 1 && len(textContent) < 75) || embedCount > 1 {
                    toRemove = true
                }

			}
		}

		if toRemove {
			doomed = append(doomed,node)
		}
	}

	for _,n := range(doomed) {
		if n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
	}

}
