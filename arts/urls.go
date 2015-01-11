package arts

// urls.go - code to look for alternate URLs within the HTML
// (ie canonical, shortlink etc...)

import (
	"code.google.com/p/cascadia"
	"fmt"
	"github.com/PuerkitoBio/purell"
	"golang.org/x/net/html"
	"net/url"
)

var urlSels = struct {
	relCanonical cascadia.Selector
	ogUrl        cascadia.Selector
	relShortlink cascadia.Selector
}{
	cascadia.MustCompile(`link[rel="canonical"]`),
	cascadia.MustCompile(`meta[property="og:url"]`),
	cascadia.MustCompile(`link[rel="shortlink"]`),
}

func sanitiseURL(link string, baseURL *url.URL) (string, error) {
	u, err := baseURL.Parse(link)
	if err != nil {
		return "", err
	}

	// we're only interested in articles, so reject obviously-not-article urls
	if u.Path == "/" || u.Path == "" {
		return "", fmt.Errorf("obviously not article")
	}

	return purell.NormalizeURL(u, purell.FlagsSafe), nil
}

// grabUrls looks for rel-canonical, og:url and rel-shortlink urls
// returns canonical url (or "") and a list of all urls (including baseURL)
func grabURLs(root *html.Node, baseURL *url.URL) (string, []string) {

	dbug := Debug.URLLogger

	canonical := ""
	all := make(map[string]struct{})

	// start with base URL
	u := purell.NormalizeURL(baseURL, purell.FlagsSafe)
	if u != "" {
		all[u] = struct{}{}
	}

	// look for canonical urls first
	for _, link := range urlSels.ogUrl.MatchAll(root) {
		txt := getAttr(link, "content")
		u, err := sanitiseURL(txt, baseURL)
		if err != nil {
			dbug.Printf("Reject og:url %s (%s)\n", txt, err)
			continue
		}

		dbug.Printf("Accept og:url %s\n", u)
		all[u] = struct{}{}
		canonical = u
	}
	for _, link := range urlSels.relCanonical.MatchAll(root) {
		txt := getAttr(link, "href")
		u, err := sanitiseURL(txt, baseURL)
		if err != nil {
			dbug.Printf("Reject rel-canonical %s (%s)\n", txt, err)
			continue
		}

		dbug.Printf("Accept rel-canonical %s\n", u)
		all[u] = struct{}{}
		canonical = u
	}

	// look for other (non-canonical) urls
	for _, link := range urlSels.relShortlink.MatchAll(root) {
		txt := getAttr(link, "href")
		u, err := sanitiseURL(getAttr(link, "href"), baseURL)
		if err != nil {
			dbug.Printf("Reject rel-shortlink %s (%s)\n", txt, err)
			continue
		}
		dbug.Printf("Accept rel-shortlink %s\n", u)
		all[u] = struct{}{}
	}

	// build up list of alternates
	allList := make([]string, 0, 8)
	for u, _ := range all {
		allList = append(allList, u)
	}

	return canonical, allList
}
