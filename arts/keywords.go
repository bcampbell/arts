package arts

// keywords.go - code to look for keywords and tags/topics in an article
// (ie <meta name="keywords"...>, rel-tag etc...)

import (
	"code.google.com/p/cascadia"
	"golang.org/x/net/html"
	//	"github.com/PuerkitoBio/purell"
	//"net/url"
	"strings"
)

var keywordSels = struct {
	meta cascadia.Selector
}{
	cascadia.MustCompile(`head meta[name="keywords"], head meta[name="news_keywords"], head meta[property="og:tags"], head meta[property="article:tag"]`),
}

func grabKeywords(root *html.Node) []Keyword {

	raw := map[string]struct{}{}

	for _, el := range keywordSels.meta.MatchAll(root) {
		for _, kw := range strings.Split(getAttr(el, "content"), ",") {
			// TODO: keep pretty-case version if dupes
			kw = strings.ToLower(strings.TrimSpace(kw))
			if kw != "" {
				raw[kw] = struct{}{}
			}
		}
	}

	out := make([]Keyword, len(raw))
	i := 0
	for kw, _ := range raw {
		out[i].Name = kw
		i++
	}
	return out
}
