package arts

import (
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"fmt"
	//"github.com/matrixik/goquery"
	"code.google.com/p/cascadia"
	"io"
	"regexp"

//	"sort"

//	"strings"
)

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
}{
	regexp.MustCompile(`(?i)byline|by-line|by_line|author|writer|credits|storycredit|firma`),
	cascadia.MustCompile("aside"),
	cascadia.MustCompile("#sidebar, #side"),
	cascadia.MustCompile("article header"),
	cascadia.MustCompile(`[itemscope][itemtype="http://schema.org/Article"]`),
	regexp.MustCompile(`(?i)comment|disqus|remark`),
}

func rateBylineContainerNode(c candidate, contentNodes []*html.Node, dbug io.Writer) {
	el := c.node()

	// TEST: likely other indicators in class/id?
	if bylineContainerPats.likelyClassPat.MatchString(getAttr(el, "class")) {
		c.addPoints(1, "indicative class")
	}
	if bylineContainerPats.likelyClassPat.MatchString(getAttr(el, "id")) {
		c.addPoints(1, "indicative id")
	}

	// TEST: inside an obvious sidebar or <aside>?
	if closest(el, bylineContainerPats.asideSel) != nil {
		c.addPoints(-3, "contained within <aside>")
	}
	if closest(el, bylineContainerPats.sidebarSel) != nil {
		c.addPoints(-3, "contained within #sidebar")
	}

	// TEST: within article container?
	//        if insideArticle(s) {
	//            c.addPoints(1,"within article container")
	//        }
	if closest(el, bylineContainerPats.articleHeaderSel) != nil {
		c.addPoints(1, "contained within <article> <header>")
	}

	// TEST: inside schema.org article?
	if closest(el, bylineContainerPats.schemaOrgArticleSel) != nil {
		c.addPoints(2, "inside schema.org article")
	}

	// TEST: within article content?
	/*	for _, contentNode := range contentNodes {
			if contains(contentNode, el) {
				c.addPoints(1, "contained within content")
			}
		}
	*/

	// TEST: at top or bottom of content?

	// TEST: share a parent with content?
	for _, contentNode := range contentNodes {
		if contains(contentNode.Parent, el) {
			c.addPoints(1, "near content")
			break
		}
	}

	// TEST: Indicative text? (eg "By...")
	if authorPats.bylineIndicativeText.MatchString(c.txt()) {
		c.addPoints(2, "indicative text")
	}
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

	// TEST: likely other indicators in parents class/id?
	/*
		if likelyClassPat.MatchString(getAttr(el.Parent, "class")) {
			c.addPoints(1, "parent has indicative class")
		}
		if likelyClassPat.MatchString(getAttr(el.Parent, "id")) {
			c.addPoints(1, "parent has indicative id")
		}
	*/

	// TEST: schema.org author
	if itemPropAuthorSel.Match(el) {
		c.addPoints(2, `itemprop="author"`)
	}

	// TEST: Indicative text? (eg "By...")
	//	if authorPats.bylineIndicativeText.MatchString(c.txt()) {
	//		c.addPoints(2, "indicative text")
	//	}

	//    TEST: looks like a name?
	nameScore := rateName(c.txt())
	if nameScore != 0 {
		c.addPoints(nameScore, "looks-like-a-name score")
	}

	// TODO:
	//  test: adjacency to headline
	//  test: adjacency to date
	//  test: penalise for full sentance text
	//  test: panalise bad urls (eg rel-tag)
	//  test: check parent for indicative text

	// TEST: likely-looking link?
	if el.DataAtom == atom.A {
		href := getAttr(el, "href")
		if goodUrlPat.MatchString(href) {
			c.addPoints(2, "likely-looking link")
		}
	}

	// TEST: inside content, but not at immediate top or bottom
	if getLinkDensity(el.Parent) < 0.75 {
		c.addPoints(-2, "in block of text")
	}
}

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

func grabAuthors(root *html.Node, contentNodes []*html.Node, dbug io.Writer) []Author {
	var authors = make(authorCandidateMap)
	var bylines = make(authorCandidateMap)

	likelyElementSel := cascadia.MustCompile("a,p,span,div,li,h3,h4,h5,h6,td,strong")
	// PASS ONE: look for any marked-up people (rel-author, hcard etc)
	for _, el := range likelyElementSel.MatchAll(root) {
		// look for structured bylines first (rel-author, hcard etc...)
		//doc.Find(`a[rel="author"], .author, .byline`).Each( func(i int, s *goquery.Selection) {

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

		authorC := newStandardCandidate(el, txt)
		containerC := newStandardCandidate(el, txt)
		rateAuthorNode(authorC, contentNodes, dbug)
		rateBylineContainerNode(containerC, contentNodes, dbug)

		if authorC.total() > 0 {
			authors[authorC.node()] = authorC
		}
		if containerC.total() > 0 {
			bylines[containerC.node()] = containerC
		}
	}

	// TODO: merge nested authors

	// PASS TWO: give containers credit for any likely-looking authors
	// they contain
	for _, byline := range bylines {
		for _, author := range authors {
			if contains(byline.node(), author.node()) {
				byline.addPoints(author.total(), fmt.Sprintf("likely-looking author (%s)", describeNode(author.node())))
			}
		}
	}

	//	sort.Sort(Reverse{candidates})

	fmt.Fprintf(dbug, "AUTHOR: %d candidates\n", len(authors))
	for _, c := range authors {
		c.dump(dbug)
	}
	fmt.Fprintf(dbug, "BYLINECONTAINERS: %d candidates\n", len(bylines))
	for _, c := range bylines {
		c.dump(dbug)
	}
	/*
		authors := make([]Author, 0, 4)
		for _, c := range candidates {
			if c.total() >= 2.0 {
				author := Author{Name: c.txt()}
				authors = append(authors, author)
			}
		}
	*/
	return make([]Author, 0)
}
