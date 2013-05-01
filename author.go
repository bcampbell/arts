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

//	"strings"
)

func grabAuthors(root *html.Node, contentNodes []*html.Node, dbug io.Writer) []Author {
	var candidates = make(CandidateList, 0, 100)

	likelyElementSel := cascadia.MustCompile("a,p,span,div,li,h3,h4,h5,h6,td,strong")
	asideSel := cascadia.MustCompile("aside")
	sidebarSel := cascadia.MustCompile("#sidebar, #side")
	articleHeaderSel := cascadia.MustCompile("article header")

	schemaOrgArticleSel := cascadia.MustCompile(`[itemscope][itemtype="http://schema.org/Article"]`)
	hentrySel := cascadia.MustCompile(".hentry")
	hcardSel := cascadia.MustCompile(".vcard")
	hcardAuthorSel := cascadia.MustCompile(".vcard.author")
	relAuthorSel := cascadia.MustCompile(`a[rel="author"]`)
	itemPropAuthorSel := cascadia.MustCompile(`[itemprop="author"]`)
	likelyClassPat := regexp.MustCompile(`(?i)byline|by-line|by_line|author|writer|credits|storycredit|firma`)

	indicativePat := regexp.MustCompile(`(?i)^\s*\b(by|text by|posted by|written by|exclusive by|reviewed by|published by|photographs by|von)\b[:]?\s*`)

	// likely-looking author urls
	goodUrlPat := regexp.MustCompile(`(?i)(^mailto:)|([/](columnistarchive|biography|profile|about|author[s]?|writer|i-author|authorinfo)[/])`)
	//    'bad_url': re.compile(r'([/](category|tag[s]?|topic[s]?|thema)[/])|(#comment[s]?$)', re.I),

	foo := likelyElementSel.MatchAll(root)
	for _, el := range foo {
		// look for structured bylines first (rel-author, hcard etc...)
		//doc.Find(`a[rel="author"], .author, .byline`).Each( func(i int, s *goquery.Selection) {

		txt := compressSpace(getTextContent(el))
		if len(txt) >= 150 {
			continue // too long
		}
		if len(txt) < 3 {
			continue // too short
		}

		c := newCandidate(el, txt)

		// TEST: marked up with hcard?
		if hcardSel.Match(el) {
			c.addScore(2, "hcard")
		}

		// TEST: hatom author?
		if hcardAuthorSel.Match(el) {
			c.addScore(2, "hatom author")
			if closest(el, hentrySel) != nil {
				c.addScore(2, "inside hentry")
			}
		}

		// TEST: rel="author"
		if relAuthorSel.Match(el) {
			c.addScore(2, "rel-author")
		}

		// TEST: likely other indicators in class/id?
		if likelyClassPat.MatchString(getAttr(el, "class")) {
			c.addScore(1, "indicative class")
		}
		if likelyClassPat.MatchString(getAttr(el, "id")) {
			c.addScore(1, "indicative id")
		}

		// TEST: likely other indicators in parents class/id?
		if likelyClassPat.MatchString(getAttr(el.Parent, "class")) {
			c.addScore(1, "parent has indicative class")
		}
		if likelyClassPat.MatchString(getAttr(el.Parent, "id")) {
			c.addScore(1, "parent has indicative id")
		}

		// TEST: schema.org author
		if itemPropAuthorSel.Match(el) {
			c.addScore(2, `itemprop="author"`)
		}

		// TEST: inside schema.org article?
		if closest(el, schemaOrgArticleSel) != nil {
			c.addScore(2, "inside schema.org article")
		}

		// TEST: Indicative text? (eg "By...")
		if indicativePat.MatchString(txt) {
			c.addScore(2, "indicative text")
		}

		// TEST: inside an obvious sidebar or <aside>?
		if closest(el, asideSel) != nil {
			c.addScore(-3, "contained within <aside>")
		}
		if closest(el, sidebarSel) != nil {
			c.addScore(-3, "contained within #sidebar")
		}

		// TEST: within article container?
		//        if insideArticle(s) {
		//            c.addScore(1,"within article container")
		//        }
		if closest(el, articleHeaderSel) != nil {
			c.addScore(1, "contained within <article> <header>")
		}

		// TEST: within article content?
		for _, contentNode := range contentNodes {
			if contains(contentNode, el) {
				c.addScore(2, "contained within content")
			}
		}

		// TODO:
		//  test: adjacency to article content
		//  test: adjacency to headline
		//  test: adjacency to date
		//  test: penalise for full sentance text
		//  test: panalise bad urls (eg rel-tag)
		//  test: check parent for indicative text

		// TEST: likely-looking link?
		if el.DataAtom == atom.A {
			href := getAttr(el, "href")
			if goodUrlPat.MatchString(href) {
				c.addScore(2, "likely-looking link")
			}

		}

		if c.TotalScore > 0 {
			candidates = append(candidates, c)
		}
	}

	sort.Sort(Reverse{candidates})

	fmt.Fprintf(dbug, "AUTHOR: %d candidates\n", len(candidates))
	if len(candidates) > 10 {
		candidates = candidates[0:10]
	}
	// show the top ten, with reasons
	for _, c := range candidates {
		c.dump(dbug)
	}

	authors := make([]Author, 0, 4)
	for _, c := range candidates {
		if c.TotalScore >= 2.0 {
			author := Author{Name: c.Txt}
			authors = append(authors, author)
		}

	}
	return authors
}
