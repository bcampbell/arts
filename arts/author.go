package arts

import (
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"fmt"
	//"github.com/matrixik/goquery"
	"code.google.com/p/cascadia"
	"regexp"
	"strings"
)

type authorCandidateMap map[*html.Node]candidate

/*
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
*/

var authorPats = struct {
	bylineIndicativeText *regexp.Regexp
	likelyClassPat       *regexp.Regexp
}{
	regexp.MustCompile(`(?i)\s*\b(by|text by|posted by|written by|exclusive by|reviewed by|report|published by|photographs by|von)\b[:]?\s*`),
	regexp.MustCompile(`(?i)name|byline|by-line|by_line|author|writer|credits|storycredit|firma`),
}

var bylineContainerPats = struct {
	likelyClassPat      *regexp.Regexp
	asideSel            cascadia.Selector
	sidebarSel          cascadia.Selector
	standfirstPat       *regexp.Regexp
	articleHeaderSel    cascadia.Selector
	schemaOrgArticleSel cascadia.Selector
	commentPat          *regexp.Regexp
	cruftIndicative     *regexp.Regexp
}{
	regexp.MustCompile(`(?i)byline|by-line|by_line|author|writer|credits|storycredit|firma`),
	cascadia.MustCompile("aside"),
	cascadia.MustCompile("#sidebar, #side"),
	regexp.MustCompile(`(?i)stand-first|standfirst|kicker|dek|articleTagline|tagline`), // also sub-heading, sub-hed, deck?
	cascadia.MustCompile("article header"),
	cascadia.MustCompile(`[itemscope][itemtype="http://schema.org/Article"]`),
	regexp.MustCompile(`(?i)\b(?:comment|disqus|livefyre|remark|conversation)\b`),
	regexp.MustCompile(`(?i)\b(?:combx|comment|community|disqus|livefyre|menu|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|promo|sponsor|shopping|tweet|twitter|facebook)\b`),
}

func rateBylineContainerNode(c candidate) {
	el := c.node()

	// TEST: inside likely cruft? (sidebars, related-articles boxes etc)
	/*
		for _, n := range parentNodes(el) {
			if bylineContainerPats.cruftIndicative.MatchString(getAttr(n, "class")) || bylineContainerPats.cruftIndicative.MatchString(getAttr(n, "id")) {
				c.addPoints(-3, fmt.Sprintf("inside cruft '%s'", describeNode(n)))
			}
		}
	*/

	elClass := getAttr(el, "class")
	elId := getAttr(el, "id")

	// TEST: is cruft itself?
	if bylineContainerPats.cruftIndicative.MatchString(getAttr(el, "class")) || bylineContainerPats.cruftIndicative.MatchString(getAttr(el, "id")) {
		c.addPoints(-3, fmt.Sprintf("looks like cruft"))
	}

	// TEST: is it a standfirst?
	if bylineContainerPats.standfirstPat.MatchString(elClass + " " + elId) {
		c.addPoints(-3, fmt.Sprintf("looks like standfirst"))
	}

	// TEST: likely other indicators in class/id?
	if bylineContainerPats.likelyClassPat.MatchString(elClass) {
		c.addPoints(1, "indicative class")
	}
	if bylineContainerPats.likelyClassPat.MatchString(elId) {
		c.addPoints(1, "indicative id")
	}

	// TEST: Indicative text? (eg "By...")
	// TODO: this test needs to be much better
	/*
		if authorPats.bylineIndicativeText.MatchString(c.txt()) {
			c.addPoints(2, "indicative text")
		}
	*/

	// TODO: TEST: contains/adjacent to date info
}

// rate node on how much it looks like an individual author
func rateAuthorNode(c candidate, contentNodes []*html.Node) {
	el := c.node()

	// TODO: handle updated uFormats: http://www.microformats.org/wiki/h-entry

	hentrySel := cascadia.MustCompile(".hentry")
	hcardSel := cascadia.MustCompile(".vcard")
	hcardAuthorSel := cascadia.MustCompile(".vcard.author")
	itemPropAuthorSel := cascadia.MustCompile(`[itemprop="author"]`)

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

	rel := strings.TrimSpace(strings.ToLower(getAttr(el, "rel")))

	// TEST: rel="author"
	if rel == "author" {
		c.addPoints(2, "rel-author")
	}
	if rel == "tag" {
		c.addPoints(-2, "rel-tag")
	}

	// TEST: likely other indicators in class/id?
	if authorPats.likelyClassPat.MatchString(getAttr(el, "class")) {
		c.addPoints(1, "indicative class")
	}
	if authorPats.likelyClassPat.MatchString(getAttr(el, "id")) {
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
	//  test: penalise if rel-tag or /category/ /topic/ whatever link
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

	// TODO: TEST: contains twitter id?
}

// TODO:
// - parse bylines ("By ... ...." etc)
// - check for bylines/dates at start of content (maybe content extraction should filter them out?)
// - better scoring on indicative text
// - de-dupe results
// - use <meta> tags to rate names
//   eg <meta name="DCSext.author" content="Martin Evans" />
// - stopwords for not-a-name list ("correspondant" etc)
func grabAuthors(root *html.Node, contentNodes []*html.Node, headlineNode *html.Node) []Author {
	dbug := Debug.AuthorsLogger
	var authors = candidateList{}
	var bylines = candidateList{}

	likelyElementSel := cascadia.MustCompile("a,p,span,div,li,h3,h4,h5,h6,td,strong")

	// get the set of elements between headline and content
	intervening := map[*html.Node]struct{}{}
	foo, err := interveningElements(headlineNode, contentNodes[0])
	if err == nil {
		for _, bar := range foo {
			intervening[bar] = struct{}{}
		}
	}

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

		authorC := newStandardCandidate(el, txt)
		containerC := newStandardCandidate(el, txt)
		if _, got := intervening[el]; got {
			authorC.addPoints(1, "between headline and content")
			containerC.addPoints(1, "between headline and content")
		}

		// any good as an author?
		rateAuthorNode(authorC, contentNodes)

		if authorC.total() > 1 {
			authors = append(authors, authorC)
		}

		// any good as a container?
		rateBylineContainerNode(containerC)
		if containerC.total() > 0 {
			bylines = append(bylines, containerC)
		}
	}

	// run over all the author candidates, and give them credit for their parents
	/*
		for _, authorC := range authors {
			descendants := authors.descendants(authorC)
			if len(descendants) > 0 {
				descendants[len(descendants)-1].addPoints(float64(len(descendants)), "likely-looking parent(s)")
			}
		}
	*/

	// discard authors which contain others
	authors = cullNestedAuthors(authors)

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

	authors.Sort()
	bylines.Sort()

	dbug.Printf("AUTHOR: %d candidates\n", len(authors))
	for _, c := range authors {
		c.dump(dbug)
	}
	dbug.Printf("BYLINECONTAINERS: %d candidates\n", len(bylines))
	for _, c := range bylines {
		c.dump(dbug)
	}

	// use top byline container
	// TODO:
	// if multiple top-scorers, check they agree.
	// if not, abort.
	if len(bylines) > 0 {
		return extractAuthors(bylines[0].node(), authors)
	}

	// nothing.
	return make([]Author, 0)
}

// cull out authors which contain others
func cullNestedAuthors(authors candidateList) candidateList {
	old := authors
	authors = make(candidateList, 0, len(old))

	for _, outer := range old {
		childCnt := 0
		for _, inner := range old {
			if contains(outer.node(), inner.node()) {
				childCnt++
			}
		}
		if childCnt == 0 {
			authors = append(authors, outer)
		}
	}
	return authors
}

func extractAuthors(container *html.Node, authors candidateList) []Author {

	extracted := make([]Author, 0)
	for _, authorC := range authors {
		if contains(container, authorC.node()) {
			a := Author{Name: authorC.txt()}
			// TODO: extract vcard stuff, email, rel-author etc etc
			extracted = append(extracted, a)
		}
	}
	return extracted
}
