package arts

import (
	"errors"
	"fmt"
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"regexp"
	"sort"
	"strings"
)

var headlinePats = struct {
	considerSel         cascadia.Selector // elements to consider
	titleSel            cascadia.Selector
	metaTitlesSel       cascadia.Selector // meta tags which have title
	itemPropHeadlineSel cascadia.Selector
	indicativePat       *regexp.Regexp
	unIndicativePat     *regexp.Regexp
}{
	cascadia.MustCompile("h1,h2,h3,h4,h5,h6,div,span,th,td"),
	cascadia.MustCompile("title"),
	cascadia.MustCompile(`meta[property="og:title"], meta[name="wp_twitter-title"]`),
	cascadia.MustCompile(`[itemprop="headline"]`),
	regexp.MustCompile(`(?i)entry-title|headline|title`),
	regexp.MustCompile(`(?i)feed-title`),
}

func grabHeadline(root *html.Node, art_url string) (string, *html.Node, error) {
	dbug := Debug.HeadlineLogger

	var candidates = make(candidateList, 0, 100)

	cookedSlug := toAlphanumeric(regexp.MustCompile("[-_]+").ReplaceAllLiteralString(getSlug(art_url), " "))
	dbug.Printf("slug: '%s'\n", cookedSlug)

	//html.Render(os.Stderr, root)
	//dumpTree(root, 0)

	var cookedTitle string
	t := headlinePats.titleSel.MatchFirst(root)
	if t != nil {
		cookedTitle = normaliseText(getTextContent(t))
	}

	// check for any interesting meta tags (og:title etc...)
	// remember that some sites append the site name to this (eg telegraph,
	// rolling stone) so we can't just take it verbatim. But it gives us clues...
	type metaTitle struct {
		cooked string
		node   *html.Node
	}
	metaTitles := []metaTitle{}
	for _, metaTitleNode := range headlinePats.metaTitlesSel.MatchAll(root) {
		cooked := normaliseText(getAttr(metaTitleNode, "content"))
		metaTitles = append(metaTitles, metaTitle{cooked, metaTitleNode})
	}

	// TODO: early-out on hatom or schema.org article

	for _, el := range headlinePats.considerSel.MatchAll(root) {

		txt := compressSpace(getTextContent(el))
		if len(txt) >= 500 {
			continue // too long
		}
		if len(txt) < 3 {
			continue // too short
		}
		cookedTxt := normaliseText(txt)
		c := newStandardCandidate(el, txt)

		// TEST: schema.org headline
		if headlinePats.itemPropHeadlineSel.Match(el) {
			c.addPoints(2, `itemprop="headline"`)
		}

		// TEST: is it a headliney element?
		tag := el.DataAtom
		if tag == atom.H1 || tag == atom.H2 || tag == atom.H3 || tag == atom.H4 {
			c.addPoints(2, "headliney")
		}
		if tag == atom.Span || tag == atom.Td {
			c.addPoints(-2, "not headliney")
		}

		// TEST: likely-looking class or id?
		cls := getAttr(el, "class")
		id := getAttr(el, "id")
		if cls != "" && (headlinePats.indicativePat.FindStringIndex(cls) != nil) {
			c.addPoints(2, "indicative class")
		}
		if id != "" && (headlinePats.indicativePat.FindStringIndex(id) != nil) {
			c.addPoints(2, "indicative id")
		}

		// TEST: unlikely-looking class or id?
		if cls != "" && (headlinePats.unIndicativePat.FindStringIndex(cls) != nil) {
			c.addPoints(-1, "unindicative class")
		}
		if id != "" && (headlinePats.unIndicativePat.FindStringIndex(id) != nil) {
			c.addPoints(-1, "unindicative id")
		}
		if len(cookedTxt) > 0 {
			// TEST: beginning of <title>?
			if strings.HasPrefix(cookedTitle, cookedTxt) {
				c.addPoints(2, "appears at start of <title>")
			}

			if wordCount(cookedTxt) >= 3 {

				// TEST: appears in page <title>?
				{
					value := jaccardWordCompare(cookedTxt, cookedTitle)
					c.addPoints((value*4)-1, "score against <title>")
				}

				// TEST: like the slug?
				alphanumericTxt := toAlphanumeric(txt)
				if cookedSlug != "" && alphanumericTxt != "" {
					value := jaccardWordCompare(alphanumericTxt, cookedSlug)
					c.addPoints((value*4)-1, "score against slug")
				}
			}

			// TEST: likely-looking meta tags
			for _, metaTitle := range metaTitles {
				value := jaccardWordCompare(cookedTxt, metaTitle.cooked)
				c.addPoints((value*6)-1, fmt.Sprintf("score against %s", describeNode(metaTitle.node)))
			}
		}

		// TEST: inside an obvious sidebar or <aside>?

		/* TODO!!!!!!!!!!!!!!!!!!!!!!!
		if s.Closest("aside").Length() > 0 {
			c.addPoints(-3, "contained within <aside>")
		}
		if s.Closest("#sidebar").Length() > 0 {
			c.addPoints(-3, "contained within #sidebar")
		}

		// TEST: within article container?
		if insideArticle(s) {
			c.addPoints(1, "within article container")
		}
		*/

		// IDEAS:
		//  promote if within <article> <header>?

		if c.total() > 0 {
			candidates = append(candidates, c)
		}
	}

	sort.Sort(Reverse{candidates})

	dbug.Printf("HEADLINE %d candidates\n", len(candidates))
	// show the top ten, with reasons
	//	if len(candidates) > 20 {
	//		candidates = candidates[0:20]
	//	}
	for _, c := range candidates {
		c.dump(dbug)
	}

	if len(candidates) > 0 {
		headline := compressSpace(headlineText(candidates[0].node()))
		return headline, candidates[0].node(), nil
	}
	return "", nil, errors.New("couldn't find a headline")
}

// get text for a headline, stripping obviously-wrong elements
func headlineText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	// sometimes have timestamps within headline...
	cls := getAttr(n, "class") + " " + getAttr(n, "id")
	if datePats.genericClasses.MatchString(cls) ||
		datePats.publishedClasses.MatchString(cls) ||
		datePats.updatedClasses.MatchString(cls) {
		return ""
	}

	txt := ""
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		txt += headlineText(child)
	}

	return txt
}
