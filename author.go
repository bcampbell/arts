package main

import (
//	"code.google.com/p/go.net/html"
//	"code.google.com/p/go.net/html/atom"
	"fmt"
	"github.com/matrixik/goquery"
//	"regexp"
	"sort"
//	"strings"
)



func extractAuthor(doc *goquery.Document) string {
    var candidates = make(CandidateList, 0, 100)

	// look for structured bylines first (rel-author, hcard etc...)
    doc.Find(`a[rel="author"], .author, .byline`).Each( func(i int, s *goquery.Selection) {
        txt := compressSpace(s.Text())
        if len(txt) >= 150 {
            return // too long
        }
        if len(txt) < 3 {
            return // too short
        }
        c := newCandidate(s.Nodes[0], txt)
		c.addScore(2,"structured markup")
        if(c.TotalScore > 0 ) {
            candidates = append(candidates, c)
        }

        // TEST: inside an obvious sidebar or <aside>?
        if s.Closest("aside").Length()>0 {
            c.addScore(-3,"contained within <aside>")
        }
        if s.Closest("#sidebar, #side").Length()>0 {
            c.addScore(-3,"contained within #sidebar")
        }

		// TEST: within article container?
        if insideArticle(s) {
            c.addScore(1,"within article container")
        }
        if s.Closest("article header").Length()>0 {
            c.addScore(1,"contained within <article> <header>")
        }

        if(c.TotalScore > 0 ) {
            candidates = append(candidates, c)
        }
	})

	// now look for unstructured bylines
//    doc.Find("a,p,span,div,li,h3,h4,h5,h6,td,strong").Each(func(i int, s *goquery.Selection) {
//	}

    sort.Sort(Reverse{candidates})
 
	fmt.Printf("AUTHOR: %d candidates\n", len(candidates))
	if( len(candidates)>10) {
		candidates = candidates[0:10]
	}
    // show the top ten, with reasons
    for _,c := range(candidates) {
        c.dump()
    }
	if len(candidates)>0 {
    return candidates[0].Txt
	}
	return ""
}

