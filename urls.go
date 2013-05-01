package arts

// urls.go - code to look for alternate URLs within the HTML
// (ie canonical, shortlink etc...)

import (
	"code.google.com/p/cascadia"
	"code.google.com/p/go.net/html"
)

var urlSels = struct {
	relCanonical cascadia.Selector
	ogUrl        cascadia.Selector
	relShortlink cascadia.Selector
}{
	cascadia.MustCompile(`head link[rel="canonical"]`),
	cascadia.MustCompile(`head meta[property="og:url"]`),
	cascadia.MustCompile(`head link[rel="shortlink"]`),
}

// grabUrls looks for rel-canonical, og:url and rel-shortlink urls
// returns canonical url (or "") and a list of alternative (non-canonical) urls
func grabUrls(root *html.Node) (string, []string) {
	canonical := ""
	all := make(map[string]bool)

	// look for canonical urls first
	for _, link := range urlSels.relCanonical.MatchAll(root) {
		url := getAttr(link, "href")
		all[url] = true
		if canonical == "" {
			canonical = url
		}
	}
	for _, link := range urlSels.ogUrl.MatchAll(root) {
		url := getAttr(link, "content")
		all[url] = true
		if canonical == "" {
			canonical = url
		}
	}

	// look for other (non-canonical) urls
	for _, link := range urlSels.relShortlink.MatchAll(root) {
		url := getAttr(link, "href")
		all[url] = true
	}

	// build up list of alternates
	delete(all, canonical)
	alternates := make([]string, 0, 8)
	for url, _ := range all {
		alternates = append(alternates, url)
	}

	return canonical, alternates
}
