package arts

import (
	"code.google.com/p/cascadia"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var headlinePats = struct {
	considerSel   cascadia.Selector // elements to consider
	titleSel      cascadia.Selector
	metaTitlesSel cascadia.Selector // meta tags which have title
}{
	cascadia.MustCompile("h1,h2,h3,h4,h5,h6,div,span,th,td"),
	cascadia.MustCompile("title"),
	cascadia.MustCompile(`meta[property="og:title"], meta[name="wp_twitter-title"]`),
}

func grabHeadline(root *html.Node, art_url string) (string, *html.Node, error) {
	dbug := Debug.HeadlineLogger

	var candidates = make(candidateList, 0, 100)

	indicative := regexp.MustCompile(`(?i)entry-title|headline|title`)

	cookedSlug := toAlphanumeric(regexp.MustCompile("[-_]+").ReplaceAllLiteralString(getSlug(art_url), " "))
	dbug.Printf("slug: '%s'\n", cookedSlug)

	//html.Render(os.Stderr, root)
	//dumpTree(root, 0)

	var cookedTitle string
	t := headlinePats.titleSel.MatchFirst(root)
	if t != nil {
		cookedTitle = toAlphanumeric(getTextContent(t))
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
		cooked := toAlphanumeric(getAttr(metaTitleNode, "content"))
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

		cookedTxt := toAlphanumeric(txt)

		c := newStandardCandidate(el, txt)

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
		if cls != "" && (indicative.FindStringIndex(cls) != nil) {
			c.addPoints(2, "indicative class")
		}

		id := getAttr(el, "id")
		if id != "" && (indicative.FindStringIndex(id) != nil) {
			c.addPoints(2, "indicative id")
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
				if cookedSlug != "" {
					value := jaccardWordCompare(cookedTxt, cookedSlug)
					c.addPoints((value*4)-1, "score against slug")
				}
			}

			// TEST: likely-looking meta tags
			for _, metaTitle := range metaTitles {
				value := jaccardWordCompare(cookedTxt, metaTitle.cooked)
				c.addPoints((value*4)-1, fmt.Sprintf("score against %s", describeNode(metaTitle.node)))
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
		return candidates[0].txt(), candidates[0].node(), nil
	}
	return "", nil, errors.New("couldn't find a headline")
}

/*
func insideArticle(s *goquery.Selection) bool {
	if s.Closest("article, #post, .article, .story-body").Length() > 0 {
		return true
	}

	return false
}
*/
