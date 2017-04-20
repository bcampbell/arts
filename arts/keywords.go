package arts

// keywords.go - code to look for keywords and tags/topics in an article
// (ie <meta name="keywords"...>, rel-tag etc...)

import (
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	//	"github.com/PuerkitoBio/purell"
	//"net/url"
	"strings"
)

var keywordSels = struct {
	meta  cascadia.Selector // for <meta> tags in head
	links cascadia.Selector // for rel-tag etc...
}{
	cascadia.MustCompile(`head meta[name="keywords"], head meta[name="news_keywords"], head meta[property="og:tags"], head meta[property="article:tag"]`),
	// HACK ALERT: .n-content-tag is specific to the FT, but same pattern as rel-tag
	// TODO: add rel-tag?
	cascadia.MustCompile(`a.n-content-tag`),
}

func grabKeywords(root *html.Node) []Keyword {

	raw := map[string]struct{}{}

	// start by looking for tags in <meta> elements
	for _, el := range keywordSels.meta.MatchAll(root) {
		for _, kw := range strings.Split(getAttr(el, "content"), ",") {
			// TODO: keep pretty-case version if dupes
			kw = strings.ToLower(strings.TrimSpace(kw))
			if kw != "" {
				raw[kw] = struct{}{}
			}
		}
	}

	// now add any link tags
	for _, el := range keywordSels.links.MatchAll(root) {
		kw := getTextContent(el)
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw != "" {
			raw[kw] = struct{}{}
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
