package arts

import (
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"fmt"
	//"github.com/matrixik/goquery"
	"code.google.com/p/cascadia"
	"io"
	"regexp"
	"sort"
	"strings"
)

type authorCandidateMap map[*html.Node]candidate

func (candidates *authorCandidateMap) accumulateScores() {
	for _, c := range *candidates {
		for _, p := range parentNodes(c.node()) {
			if parentC, got := (*candidates)[p]; got {
				parentC.addPoints(c.total(), fmt.Sprintf("likely-looking child (%s)", describeNode(c.node())))
			}
		}
	}
}

func (candidates authorCandidateMap) findParents(c candidate) (out []candidate) {
	n := c.node().Parent
	for n != nil {
		if parentC, got := (candidates)[n]; got {
			out = append(out, parentC)
		}
		n = n.Parent
	}
	return
}

func (candidates authorCandidateMap) descendants(c candidate) []candidate {
	out := make([]candidate, 0)

	walkChildren(c.node(), func(n *html.Node) {
		if descendant, got := candidates[n]; got {
			out = append(out, descendant)
		}
	})
	return out
}

var authorPats = struct {
	bylineIndicativeText *regexp.Regexp
}{
	regexp.MustCompile(`(?i)^\s*\b(by|text by|posted by|written by|exclusive by|reviewed by|report|published by|photographs by|von)\b[:]?\s*`),
}

var bylineContainerPats = struct {
	likelyClassPat      *regexp.Regexp
	asideSel            cascadia.Selector
	sidebarSel          cascadia.Selector
	articleHeaderSel    cascadia.Selector
	schemaOrgArticleSel cascadia.Selector
	commentPat          *regexp.Regexp
	cruftIndicative     *regexp.Regexp
}{
	regexp.MustCompile(`(?i)byline|by-line|by_line|author|writer|credits|storycredit|firma`),
	cascadia.MustCompile("aside"),
	cascadia.MustCompile("#sidebar, #side"),
	cascadia.MustCompile("article header"),
	cascadia.MustCompile(`[itemscope][itemtype="http://schema.org/Article"]`),
	regexp.MustCompile(`(?i)comment|disqus|livefyre|remark|conversation`),
	regexp.MustCompile(`(?i)combx|comment|community|disqus|livefyre|menu|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|promo|sponsor|shopping|tweet|twitter`),
}

func rateBylineContainerNode(c candidate, contentNodes []*html.Node, headlineNode *html.Node, dbug io.Writer) {
	el := c.node()

	// TEST: inside likely cruft? (sidebars, related-articles boxes etc)
	for _, n := range parentNodes(el) {
		if bylineContainerPats.cruftIndicative.MatchString(getAttr(n, "class")) || bylineContainerPats.cruftIndicative.MatchString(getAttr(n, "id")) {
			c.addPoints(-3, fmt.Sprintf("inside cruft '%s'", describeNode(n)))
		}
	}

	// TEST: likely other indicators in class/id?
	if bylineContainerPats.likelyClassPat.MatchString(getAttr(el, "class")) {
		c.addPoints(1, "indicative class")
	}
	if bylineContainerPats.likelyClassPat.MatchString(getAttr(el, "id")) {
		c.addPoints(1, "indicative id")
	}

	// TEST: within article container?
	if closest(el, bylineContainerPats.articleHeaderSel) != nil {
		c.addPoints(1, "contained within <article> <header>")
	}

	// TEST: inside schema.org article?
	if closest(el, bylineContainerPats.schemaOrgArticleSel) != nil {
		c.addPoints(1, "inside schema.org article")
	}

	// TEST: proximity to headline
	if headlineNode != nil {
		interveningChars := 0
		n := el
		for {
			n = prevNode(n)
			if n == nil {
				break
			}
			if n == headlineNode {
				if interveningChars == 0 {
					c.addPoints(2, "adjacent to headline")
				}
				break
			}
			if n.Type == html.TextNode {
				s := strings.TrimSpace(n.Data)
				interveningChars += len(s)
			}
		}
	}

	// TEST: Indicative text? (eg "By...")
	if authorPats.bylineIndicativeText.MatchString(c.txt()) {
		c.addPoints(2, "indicative text")
	}

	// TODO: TEST: contains twitter id?
}

// rate node on how much it looks like an individual author
func rateAuthorNode(c candidate, contentNodes []*html.Node, dbug io.Writer) {
	el := c.node()

	hentrySel := cascadia.MustCompile(".hentry")
	hcardSel := cascadia.MustCompile(".vcard")
	hcardAuthorSel := cascadia.MustCompile(".vcard.author")
	relAuthorSel := cascadia.MustCompile(`a[rel="author"]`)
	itemPropAuthorSel := cascadia.MustCompile(`[itemprop="author"]`)
	likelyClassPat := regexp.MustCompile(`(?i)byline|by-line|by_line|author|writer|credits|storycredit|firma`)

	// likely-looking author urls
	goodUrlPat := regexp.MustCompile(`(?i)(^mailto:)|([/](columnistarchive|biography|profile|about|author[s]?|writer|i-author|authorinfo)[/])`)
	//    'bad_url': re.compile(r'([/](category|tag[s]?|topic[s]?|thema)[/])|(#comment[s]?$)', re.I),

	// TEST: marked up with hcard?
	if hcardSel.Match(el) {
		c.addPoints(2, "hcard")
	}

	// TEST: hatom author?
	if hcardAuthorSel.Match(el) {
		c.addPoints(2, "hatom author")
		if closest(el, hentrySel) != nil {
			c.addPoints(2, "inside hentry")
		}
	}

	// TEST: rel="author"
	if relAuthorSel.Match(el) {
		c.addPoints(2, "rel-author")
	}

	// TEST: likely other indicators in class/id?
	if likelyClassPat.MatchString(getAttr(el, "class")) {
		c.addPoints(1, "indicative class")
	}
	if likelyClassPat.MatchString(getAttr(el, "id")) {
		c.addPoints(1, "indicative id")
	}

	// TEST: schema.org author
	if itemPropAuthorSel.Match(el) {
		c.addPoints(2, `itemprop="author"`)
	}

	//    TEST: looks like a name?
	nameScore := rateName(c.txt())
	if nameScore != 0 {
		c.addPoints(nameScore, "looks-like-a-name score")
	}

	// TODO:
	//  test: penalise for full sentence text (eg punctuation)
	//  test: penalise for stopwords ("about" etc)
	//  test: penalise if rel-tag
	//  test: check for adjacent indicative text

	// TEST: likely-looking link?
	if el.DataAtom == atom.A {
		href := getAttr(el, "href")
		if goodUrlPat.MatchString(href) {
			c.addPoints(2, "likely-looking link")
		}
	}

	// TEST: inside content, but not at immediate top or bottom
	//	if getLinkDensity(el.Parent) < 0.75 {
	//		c.addPoints(-2, "in block of text")
	//	}
}

func grabAuthors(root *html.Node, contentNodes []*html.Node, headlineNode *html.Node, dbug io.Writer) []Author {
	var authors = make(authorCandidateMap)
	var bylines = make(authorCandidateMap)

	likelyElementSel := cascadia.MustCompile("a,p,span,div,li,h3,h4,h5,h6,td,strong")
	// PASS ONE: look for any marked-up people (rel-author, hcard etc)

	// looking for:
	//  - elements containing individual authors
	//  - elements that look like byline containers
	for _, el := range likelyElementSel.MatchAll(root) {
		earlyOut := false
		txt := compressSpace(getTextContent(el))
		if len(txt) >= 150 {
			earlyOut = true
		} else if len(txt) < 3 {
			earlyOut = true
		} else {
			// inside comment?
			// if so, just ignore.
			for _, parent := range parentNodes(el) {
				if bylineContainerPats.commentPat.MatchString(getAttr(parent, "class")) {
					earlyOut = true
				}
				if bylineContainerPats.commentPat.MatchString(getAttr(parent, "id")) {
					earlyOut = true
				}
			}
		}

		if earlyOut {
			continue
		}

		// any good as an author?
		authorC := newStandardCandidate(el, txt)
		rateAuthorNode(authorC, contentNodes, dbug)

		if authorC.total() >= 1 {
			authors[authorC.node()] = authorC
		}

		// any good as a container?
		containerC := newStandardCandidate(el, txt)
		rateBylineContainerNode(containerC, contentNodes, headlineNode, dbug)
		if containerC.total() > 0 {
			bylines[containerC.node()] = containerC
		}
	}

	// run over all the author candidates, and give them credit for their parents
	for _, authorC := range authors {
		descendants := authors.descendants(authorC)
		if len(descendants) > 0 {
			descendants[len(descendants)-1].addPoints(float64(len(descendants)), "likely-looking parent(s)")
		}
	}

	// PASS TWO: give containers credit for containing likely-looking authors
	for _, byline := range bylines {
		cnt := 0
		for _, author := range authors {
			if byline.node() == author.node() {
				//byline.addPoints(1, "also a likely-looking author")
			} else if contains(byline.node(), author.node()) {
				cnt += 1
			}
		}
		if cnt > 0 {
			byline.addPoints(1, fmt.Sprintf("contains likely-looking author(s)"))
		}
	}

	// TODO:
	//  if no containers, promote best author
	//  merge nested containers?

	// extract authors inside best container

	ranked := make(candidateList, len(bylines))
	i := 0
	for _, c := range bylines {
		ranked[i] = c
		i++
	}
	sort.Sort(Reverse{ranked})

	fmt.Fprintf(dbug, "AUTHOR: %d candidates\n", len(authors))
	for _, c := range authors {
		c.dump(dbug)
	}
	fmt.Fprintf(dbug, "BYLINECONTAINERS: %d candidates\n", len(ranked))
	for _, c := range ranked {
		c.dump(dbug)
	}

	if len(ranked) > 0 {
		return extractAuthors(ranked[0], authors)
	}
	return make([]Author, 0)
}

func extractAuthors(container candidate, authorCandidates authorCandidateMap) []Author {
	extracted := make([]Author, 0)
	for _, authorC := range authorCandidates {
		if contains(container.node(), authorC.node()) {
			a := Author{Name: authorC.txt()}
			// TODO: extract vcard stuff, email, rel-author etc etc
			extracted = append(extracted, a)
		}
	}
	return extracted
}
