package arts

// urls.go - code to look for alternate URLs within the HTML
// (ie canonical, shortlink etc...)

import (
	"code.google.com/p/cascadia"
	"github.com/PuerkitoBio/purell"
	"golang.org/x/net/html"
	"net/url"
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

func sanitiseURL(link string, baseURL *url.URL) (string, error) {
	u, err := baseURL.Parse(link)
	if err != nil {
		return "", err
	}

	return purell.NormalizeURL(u, purell.FlagsSafe), nil
}

// grabUrls looks for rel-canonical, og:url and rel-shortlink urls
// returns canonical url (or "") and a list of all urls (including baseURL)
func grabURLs(root *html.Node, baseURL *url.URL) (string, []string) {

	canonical := ""
	all := make(map[string]bool)

	// start with base URL
	u := purell.NormalizeURL(baseURL, purell.FlagsSafe)
	if u != "" {
		all[u] = true
	}

	// look for canonical urls first
	for _, link := range urlSels.ogUrl.MatchAll(root) {
		u, err := sanitiseURL(getAttr(link, "content"), baseURL)
		if err != nil {
			continue
		}

		all[u] = true
		canonical = u
	}
	for _, link := range urlSels.relCanonical.MatchAll(root) {
		u, err := sanitiseURL(getAttr(link, "href"), baseURL)
		if err != nil {
			continue
		}

		all[u] = true
		canonical = u
	}

	// look for other (non-canonical) urls
	for _, link := range urlSels.relShortlink.MatchAll(root) {
		u, err := sanitiseURL(getAttr(link, "href"), baseURL)
		if err != nil {
			continue
		}
		all[u] = true
	}

	// build up list of alternates
	allList := make([]string, 0, 8)
	for u, _ := range all {
		allList = append(allList, u)
	}

	return canonical, allList
}
