package arts

// content.go is concerned with extracting the main text content of an
// online article.
// It started off as a direct port of arc90's readbility javascript, to
// the point of keeping a lot of the original comments, variable and
// function names.
// (see http://code.google.com/p/arc90labs-readability )
// It diverges a little because readability is meant to be used as part of
// a browser extension, with all the DOM structure that entails, whereas
// this version is designed to be used on a bare parsed html.Node tree.

import (
	"code.google.com/p/cascadia"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	//	"github.com/matrixik/goquery"
	"math"
	"regexp"
	//	"sort"
	"strings"
)

// todo: define a specialised type for content candidates?
// candidateMap stores candidates for quick lookup by node
type candidateMap map[*html.Node]candidate

// get returns an existing candidiate struct or create a blank new one
func (candidates candidateMap) get(n *html.Node) candidate {
	c, ok := candidates[n]
	if !ok {
		c = newStandardCandidate(n, "")
		candidates[n] = c
	}
	return c
}

// assign initial scoring to a potential content candidate
func initializeNode(c candidate) {
	switch c.node().DataAtom {
	case atom.Article:
		c.addPoints(8, "<article>")
	case atom.Div:
		c.addPoints(5, "<div>")
	case atom.Pre, atom.Td, atom.Blockquote:
		c.addPoints(3, "<pre>, <td> or <blockquote>")
	case atom.Address, atom.Ol, atom.Ul, atom.Dl, atom.Dd, atom.Li, atom.Form:
		c.addPoints(-3, "address, list or form")
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Th:
		c.addPoints(-5, "heading")
	}

	if score := getClassWeight(c.node()); score != 0 {
		c.addPoints(score, "class/id score")
	}
}

// remove all <script> elements
func removeScripts(root *html.Node) {
	sel := cascadia.MustCompile("script")
	for _, script := range sel.MatchAll(root) {
		script.Parent.RemoveChild(script)
	}
}

var unlikelyCandidates = regexp.MustCompile(`(?i)combx|comment|community|disqus|livefyre|extra|foot|header|menu|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|tweet|twitter`)
var okMaybeItsACandidate = regexp.MustCompile(`(?i)and|article|body|column|main|shadow`)

var positivePat = regexp.MustCompile(`(?i)article|body|content|entry|hentry|main|page|pagination|post|text|blog|story`)
var negativePat = regexp.MustCompile(`(?i)combx|comment|com-|contact|foot|footer|footnote|masthead|media|meta|outbrain|promo|related|scroll|shoutbox|sidebar|sponsor|shopping|tags|tool|widget`)

// grabContent finds the nodes in the page which contain the actual article text.
// Returns a slice of node pointers (in order), and a map containing all
// the content scores calculated. The scores can be used in a later pass to help
// remove cruft nodes in the text (eg share/like buttons etc)
func grabContent(root *html.Node) ([]*html.Node, candidateMap) {
	dbug := Debug.ContentLogger
	var candidates = make(candidateMap)

	stripUnlikelyCandidates := false

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

			// TODO: this lets through things it shouldn't, eg ".dsq-comment-body"
			if unlikelyCandidates.MatchString(unlikelyMatchString) == true &&
				okMaybeItsACandidate.MatchString(unlikelyMatchString) == false &&
				node.DataAtom != atom.Body {
				dbug.Printf("Removing unlikely candidate - %s\n", describeNode(node))
				node.Parent.RemoveChild(node)
				continue
			}
		}

		if node.DataAtom == atom.P || node.DataAtom == atom.Td || node.DataAtom == atom.Pre {
			nodesToScore = append(nodesToScore, node)
		}
		/* XYZZY TODO: Turn all divs that don't have children block level elements into p's */
		// concrete example: http://www.theatlantic.com/national/archive/2013/04/the-boston-marathon-bombing-keep-calm-and-carry-on/275014/
		// all paras are divs.
	}

	dbug.Printf("%d nodes to score\n", len(nodesToScore))

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
			initializeNode(candidates.get(parentNode))
		}
		if grandParentNode != nil {
			if _, exists := candidates[grandParentNode]; !exists {
				initializeNode(candidates.get(grandParentNode))
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
		candidates.get(parentNode).addPoints(contentScore, "Child content")
		if grandParentNode != nil {
			halfScore := contentScore / 2
			if halfScore > 0 {
				candidates.get(grandParentNode).addPoints(halfScore, "Child content")
			}
		}
	}

	contentNodes := make([]*html.Node, 0, 64)
	if len(candidates) == 0 {
		// oh.
		dbug.Printf("no candidates\n")
		return contentNodes, candidates
	}

	/**
	 * Scale the final candidates score based on link density. Good content should have a
	 * relatively small link density (5% or less) and be mostly unaffected by this operation.
	 **/
	for _, c := range candidates {
		c.scalePoints((1 - getLinkDensity(c.node())), "link density")
	}

	/**
	 * After we've calculated scores, loop through all of the possible candidate nodes we found
	 * and find the one with the highest score.
	**/
	var topCandidate candidate = nil
	for _, c := range candidates {
		if topCandidate == nil || c.total() > topCandidate.total() {
			topCandidate = c
		}
	}

	dbug.Printf(" %d candidates:\n", len(candidates))
	for _, c := range candidates {
		dbug.Printf("  %f: %s\n", c.total(), describeNode(c.node()))
		c.dump(dbug)
	}
	//html.Render(os.Stdout, topCandidate.node())

	/**
	 * Now that we have the top candidate, look through its siblings for content that might also be related.
	 * Things like preambles, content split by ads that we removed, etc.
	**/

	dbug.Printf("picked %s (score %f)\n", describeNode(topCandidate.node()), topCandidate.total())
	siblingScoreThreshold := topCandidate.total() * 0.2
	if siblingScoreThreshold < 10 {
		siblingScoreThreshold = 10
	}

	for siblingNode := topCandidate.node().Parent.FirstChild; siblingNode != nil; siblingNode = siblingNode.NextSibling {
		useIt := false
		if siblingNode == topCandidate.node() {
			useIt = true
		} else {

			contentBonus := 0.0
			/* Give a bonus if sibling nodes and top candidates have the exact same classname */
			topClass := getAttr(topCandidate.node(), "class")
			if getAttr(siblingNode, "class") == topClass && topClass != "" {
				contentBonus += topCandidate.total() * 0.2
			}
			if sc, ok := candidates[siblingNode]; ok == true && sc.total()+contentBonus >= siblingScoreThreshold {
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

	dbug.Printf("got %d content nodes:\n", len(contentNodes))
	for _, n := range contentNodes {
		dbug.Printf("  %s\n", describeNode(n))
	}

	return contentNodes, candidates
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

// Remove all extraneous crap in the content - related articles, share buttons etc...
// (equivalent to prepArticle() in readbility.js)
func removeCruft(contentNodes []*html.Node, candidates candidateMap) {
	dbug := Debug.ContentLogger
	dbug.Printf("Cruft removal\n")

	zapConditionally(contentNodes, "form", candidates)
	zap(contentNodes, "object")
	zap(contentNodes, "h1")

	// If there is only one h2, they are probably using it
	// as a header and not a subheader, so remove it since we already have a header.
	h2Count := 0
	h2Sel := cascadia.MustCompile("h2")
	for _, node := range contentNodes {
		h2Count += len(h2Sel.MatchAll(node))
	}

	if h2Count == 1 {
		zap(contentNodes, "h2")
	}
	zap(contentNodes, "iframe")

	//cleanHeaders()

	/* Do these last as the previous stuff may have removed junk that will affect these */
	zapConditionally(contentNodes, "table", candidates)
	zapConditionally(contentNodes, "ul", candidates)
	zapConditionally(contentNodes, "div", candidates)
}

func zap(contentNodes []*html.Node, tagSel string) {
	doomed := make([]*html.Node, 0, 32)
	sel := cascadia.MustCompile(tagSel)
	for _, contentNode := range contentNodes {
		for _, node := range sel.MatchAll(contentNode) {
			// XYZZY TODO: preserve videos?
			doomed = append(doomed, node)
		}
	}
	for _, n := range doomed {
		if n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
	}
}

/**
 * Clean a set of elements, removing all matching tags if they look fishy.
 * "Fishy" is an algorithm based on content length, classnames, link density, number of images & embeds, etc.
 **/
func zapConditionally(contentNodes []*html.Node, tagSel string, candidates candidateMap) {
	dbug := Debug.ContentLogger

	doomed := make([]*html.Node, 0, 32)
	sel := cascadia.MustCompile(tagSel)
	for _, e := range contentNodes {
		for _, node := range sel.MatchAll(e) {
			weight := getClassWeight(node)
			var contentScore float64 = 0.0
			if c, ok := candidates[node]; ok {
				contentScore = c.total()
			}
			toRemove := false

			if weight+contentScore < 0 {
				dbug.Printf("kill %s: weight + contentScore < 0\n", describeNode(node))
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

					if img > p {
						dbug.Printf("kill %s: more images than paras\n", describeNode(node))
						toRemove = true
					} else if li > p && node.DataAtom != atom.Ul && node.DataAtom != atom.Ol {
						dbug.Printf("kill %s: more <li>s than paras\n", describeNode(node))
						toRemove = true
					} else if input > int(math.Floor(float64(p)/3.0)) {
						dbug.Printf("kill %s: too many <input>s\n", describeNode(node))
						toRemove = true
					} else if len(textContent) < 25 && (img == 0 || img > 2) {
						dbug.Printf("kill %s: too little text\n", describeNode(node))
						toRemove = true
					} else if weight < 25 && linkDensity > 0.2 {
						dbug.Printf("kill %s: link density too high\n", describeNode(node))
						toRemove = true
					} else if weight >= 25 && linkDensity > 0.5 {
						dbug.Printf("kill %s: link density too high 2\n", describeNode(node))
						toRemove = true
					} else if (embedCount == 1 && len(textContent) < 75) || embedCount > 1 {
						dbug.Printf("kill %s: embedCount\n", describeNode(node))
						toRemove = true
					}

				}
			}

			if toRemove {
				doomed = append(doomed, node)
			}
		}
	}

	for _, n := range doomed {
		if n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
	}
}

// Tidy up extracted content into something that'll produce reasonable html when
// rendered
// - remove comments
// - trim whitespace
// - remove non-essential attrs (TODO: still some more to do on this)
// - TODO make links absolute
func sanitiseContent(contentNodes []*html.Node) {

	for _, node := range contentNodes {
		tidyNode(node)
	}
}
