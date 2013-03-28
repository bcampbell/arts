package main

import (
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"fmt"
	"github.com/matrixik/goquery"
	"regexp"
	"sort"
	"strings"
)

type Candidate struct {
	node  *html.Node
	txt   string
	score int
}
type Candidates []Candidate

func (s Candidates) Len() int           { return len(s) }
func (s Candidates) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Candidates) Less(i, j int) bool { return s[i].score < s[j].score }

type Reverse struct {
	sort.Interface
}

func (r Reverse) Less(i, j int) bool {
	return r.Interface.Less(j, i)
}


func extract_headline(doc *goquery.Document, art_url string) string {

	var candidates Candidates


	indicative := regexp.MustCompile(`(?i)entry-title|headline|title`)

    cooked_slug := toAlphanumeric(regexp.MustCompile("[-_]+").ReplaceAllLiteralString(getSlug(art_url), ""))

	cooked_title := toAlphanumeric(doc.Find("head title").Text())
    og_title,foo := doc.Find(`head meta[property="og:title"]`).Attr("content")
    var cooked_og_title string
    if foo {
        cooked_og_title = toAlphanumeric(og_title)
    }

    // TODO: early-out on hatom or schema.org article
    // but not opengraph og:title (eg telegraph appends " - Telegraph",
    // rolling stone does similar others are bould to)

	doc.Find("h1,h2,h3,h4,h5,h6,div,span,th,td").Each(func(i int, s *goquery.Selection) {
		//doc.Find("h1,h2,h3,h4,h5,h6").Each(func(i int, s *goquery.Selection) {

		var score int = 0
		txt := compressSpace(s.Text())
		if len(txt) >= 500 {
			return // too long
		}
		if len(txt) < 3 {
			return // too short
		}
        cooked_txt := toAlphanumeric(txt)
		//        fmt.Printf("%d  '%s'\n", s.Length(), txt)

        dbug("check <%s> '%s'\n", s.Nodes[0].DataAtom.String(), txt)

		// TEST: is it a headliney element?
		tag := s.Nodes[0].DataAtom
		if tag == atom.H1 || tag == atom.H2 || tag == atom.H3 || tag == atom.H4 {
			score += 2
			dbug("  +2 headliney\n")
		}
		if tag == atom.Span || tag == atom.Td {
			score -= 2
			dbug("  -2 not headliney\n")
		}

		// TEST: likely-looking class or id?
		cls, foo := s.Attr("class")
		if foo && (indicative.FindStringIndex(cls) != nil) {
			//logging.debug("  likely class")
			score += 2
			dbug("  +2 indicative class\n")
		}

		id, foo := s.Attr("id")
		if foo && (indicative.FindStringIndex(id) != nil) {
			//logging.debug("  likely id")
			score += 2
			dbug("  +2 indicative id\n")
		}

		// TEST: appears in page <title>?
		if strings.Contains(cooked_title, cooked_txt) {
			score += 3
			dbug("  +3 appears in <title>\n")
		}

        // TEST: appears in og:title?
        if strings.Contains(cooked_og_title, cooked_txt) {
			score += 3
			dbug("  +3 appears in og:title\n")
        }

        // TEST: appears in slug?
        {
            var matches int = 0;
            parts := strings.Split(cooked_txt, " ")
            if len(parts)>1 {
                for _,part := range parts {
                    if strings.Contains(cooked_slug, part) {
                        matches += 1
                    }
                }
                var value int = (5*matches) / len(parts)
                if value > 0  {
                    score += value
                    dbug("  +%d match slug", value)
                }
            }
        }

		// TODO:
		// TEST: does it appear in likely looking <meta> tags? "Headline" etc...

		candidates = append(candidates, Candidate{s.Nodes[0], txt, score})
	})

	sort.Sort(Reverse{candidates})
	fmt.Printf("CANDIDATES:\n %v\n", candidates[0:5])

	return candidates[0].txt
}

