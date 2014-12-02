package arts

// keywords.go - code to look for keywords and tags/topics in an article
// (ie <meta name="keywords"...>, rel-tag etc...)

import (
	"code.google.com/p/cascadia"
	"golang.org/x/net/html"
	//	"github.com/PuerkitoBio/purell"
	"net/url"
	"strings"
)

var publicationPats = struct {
	siteNameSel cascadia.Selector
}{
	cascadia.MustCompile(`head meta[property="og:site_name"], head meta[name="twitter:domain"]`),
}

// <meta property="og:site_name" content="The Daily Blah" />
// <meta name="twitter:domain" content="The Daily Blah"/>

func grabPublication(root *html.Node, art *Article) Publication {
	// TODO: check og:site_name and other metadata
	pub := Publication{}

	// get domain
	bestURL := art.BestURL()
	if bestURL != "" {
		u, err := url.Parse(bestURL)
		if err == nil {
			pub.Domain = u.Host
		}
	}

	// get name of site
	el := publicationPats.siteNameSel.MatchFirst(root)
	if el != nil {
		pub.Name = strings.TrimSpace(getAttr(el, "content"))
	}
	return pub
}
