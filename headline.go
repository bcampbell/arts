package arts

import (
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"errors"
	"github.com/matrixik/goquery"
	"regexp"
	"sort"
	"strings"
)

// TODO: phase out goquery - just use cascadia directly

func grabHeadline(root *html.Node, art_url string) (string, *html.Node, error) {
	dbug := Debug.HeadlineLogger
	doc := goquery.NewDocumentFromNode(root)

	var candidates = make(candidateList, 0, 100)

	indicative := regexp.MustCompile(`(?i)entry-title|headline|title`)

	cooked_slug := toAlphanumeric(regexp.MustCompile("[-_]+").ReplaceAllLiteralString(getSlug(art_url), ""))

	cooked_title := toAlphanumeric(doc.Find("head title").Text())
	og_title, foo := doc.Find(`head meta[property="og:title"]`).Attr("content")
	var cooked_og_title string
	if foo {
		cooked_og_title = toAlphanumeric(og_title)
	}

	// TODO: early-out on hatom or schema.org article
	// but not opengraph og:title (eg telegraph appends " - Telegraph",
	// rolling stone does similar, others are bound to too)

	doc.Find("h1,h2,h3,h4,h5,h6,div,span,th,td").Each(func(i int, s *goquery.Selection) {
		//doc.Find("h1,h2,h3,h4,h5,h6").Each(func(i int, s *goquery.Selection) {

		txt := compressSpace(s.Text())
		if len(txt) >= 500 {
			return // too long
		}
		if len(txt) < 3 {
			return // too short
		}

		cooked_txt := toAlphanumeric(txt)

		c := newStandardCandidate(s.Nodes[0], txt)

		// TEST: is it a headliney element?
		tag := s.Nodes[0].DataAtom
		if tag == atom.H1 || tag == atom.H2 || tag == atom.H3 || tag == atom.H4 {
			c.addPoints(2, "headliney")
		}
		if tag == atom.Span || tag == atom.Td {
			c.addPoints(-2, "not headliney")
		}

		// TEST: likely-looking class or id?
		cls, foo := s.Attr("class")
		if foo && (indicative.FindStringIndex(cls) != nil) {
			c.addPoints(2, "indicative class")
		}

		id, foo := s.Attr("id")
		if foo && (indicative.FindStringIndex(id) != nil) {
			c.addPoints(2, "indicative id")
		}

		if len(cooked_txt) > 0 {
			if wordCount(cooked_txt) >= 3 {
				// TEST: appears in page <title>?
				if strings.Contains(cooked_title, cooked_txt) {
					c.addPoints(3, "appears in <title>")
				}

				// TEST: appears in og:title?
				if strings.Contains(cooked_og_title, cooked_txt) {
					c.addPoints(3, "appears in og:title")
				}

				// TEST: appears in slug?
				var matches int = 0
				parts := strings.Split(cooked_txt, " ")
				if len(parts) > 1 {
					for _, part := range parts {
						if strings.Contains(cooked_slug, part) {
							matches += 1
						}
					}
					var value float64 = float64(5*matches) / float64(len(parts))
					if value > 0 {
						c.addPoints(value, "match slug")
					}
				}
			}
		}

		// TODO:
		// TEST: does it appear in likely looking <meta> tags? "Headline" etc...

		// TEST: inside an obvious sidebar or <aside>?
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

		// IDEAS:
		//  promote if within <article> <header>?

		if c.total() > 0 {
			candidates = append(candidates, c)
		}
	})

	sort.Sort(Reverse{candidates})

	dbug.Printf("HEADLINE %d candidates\n", len(candidates))
	// show the top ten, with reasons
	if len(candidates) > 10 {
		candidates = candidates[0:10]
	}
	for _, c := range candidates {
		c.dump(dbug)
	}

	if len(candidates) > 0 {
		return candidates[0].txt(), candidates[0].node(), nil
	}
	return "", nil, errors.New("couldn't find a headline")
}

func insideArticle(s *goquery.Selection) bool {
	if s.Closest("article, #post, .article, .story-body").Length() > 0 {
		return true
	}

	return false
}
