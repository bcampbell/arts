package discover

//
//
// TODO:
//   should be able to guess article link format statistically
//   handle/allow subdomains (eg: www1.politicalbetting.com)
//   filter unwanted navlinks (eg "mirror.co.uk/all-about/fred bloggs")
//   HTTP error handling
//   multiple url formats (eg spectator has multiple cms's)
//   logging

import (
	"code.google.com/p/cascadia"
	"golang.org/x/net/html"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

type NullLogger struct{}

func (l NullLogger) Printf(format string, v ...interface{}) {
}

type DiscovererDef struct {
	Name   string
	URL    string
	ArtPat []string
	NavSel string
	// BaseErrorThreshold is starting number of http errors to accept before
	// bailing out.
	// error threshold formula: base + 10% of successful request count
	BaseErrorThreshold int

	// Hostpat is a regex matching accepted domains
	// if empty, reject everything on a different domain
	HostPat string

	// If NoStripQuery is set then article URLs won't have the query part zapped
	NoStripQuery bool
}

type DiscoverStats struct {
	ErrorCount int
	FetchCount int
}

type Discoverer struct {
	Name               string
	StartURL           url.URL
	ArtPats            []*regexp.Regexp
	NavLinkSel         cascadia.Selector
	BaseErrorThreshold int
	StripFragments     bool
	StripQuery         bool
	HostPat            *regexp.Regexp

	ErrorLog Logger
	InfoLog  Logger
	Stats    DiscoverStats
}

func NewDiscoverer(cfg DiscovererDef) (*Discoverer, error) {
	disc := &Discoverer{}
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}
	disc.Name = cfg.Name
	disc.StartURL = *u
	disc.ArtPats = make([]*regexp.Regexp, 0, len(cfg.ArtPat))
	for _, pat := range cfg.ArtPat {
		re, err := regexp.Compile(pat)
		if err != nil {
			return nil, err
		}
		disc.ArtPats = append(disc.ArtPats, re)
	}

	if cfg.NavSel == "" {
		disc.NavLinkSel = nil
	} else {
		sel, err := cascadia.Compile(cfg.NavSel)
		if err != nil {
			return nil, err
		}
		disc.NavLinkSel = sel
	}
	disc.BaseErrorThreshold = cfg.BaseErrorThreshold

	if cfg.HostPat != "" {
		re, err := regexp.Compile(cfg.HostPat)
		if err != nil {
			return nil, err
		}
		disc.HostPat = re
	}

	// defaults
	disc.StripFragments = true
	disc.StripQuery = !cfg.NoStripQuery
	disc.ErrorLog = NullLogger{}
	disc.InfoLog = NullLogger{}
	return disc, nil
}

func (disc *Discoverer) Run(client *http.Client) (LinkSet, error) {
	// reset stats
	disc.Stats = DiscoverStats{}

	queued := make(LinkSet) // nav pages to scan for article links
	seen := make(LinkSet)   // nav pages we've scanned
	arts := make(LinkSet)   // article links we've found so far

	queued.Add(disc.StartURL)

	for len(queued) > 0 {
		pageURL := queued.Pop()
		seen.Add(pageURL)
		//

		root, err := disc.fetchAndParse(client, &pageURL)
		if err != nil {
			disc.ErrorLog.Printf("%s\n", err.Error())
			disc.Stats.ErrorCount++
			if disc.Stats.ErrorCount > disc.BaseErrorThreshold+(disc.Stats.FetchCount/10) {
				return nil, errors.New("Error threshold exceeded")
			} else {
				continue
			}
		}
		disc.Stats.FetchCount++

		navLinks, err := disc.findNavLinks(root)
		if err != nil {
			return nil, err
		}
		for navLink, _ := range navLinks {
			if _, got := seen[navLink]; !got {
				queued.Add(navLink)
			}
		}

		foo, err := disc.findArticles(&pageURL, root)
		if err != nil {
			return nil, err
		}
		arts.Merge(foo)

		disc.InfoLog.Printf("Visited %s, found %d articles\n", pageURL.String(), len(foo))
	}

	return arts, nil
}

func (disc *Discoverer) fetchAndParse(client *http.Client, pageURL *url.URL) (*html.Node, error) {
	resp, err := client.Get(pageURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = errors.New(fmt.Sprintf("HTTP code %d (%s)", resp.StatusCode, pageURL.String()))

		return nil, err

	}

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return root, nil
}

var aSel cascadia.Selector = cascadia.MustCompile("a")

func (disc *Discoverer) findArticles(baseURL *url.URL, root *html.Node) (LinkSet, error) {
	arts := make(LinkSet)
	for _, a := range aSel.MatchAll(root) {
		u, err := disc.CookArticleURL(baseURL, GetAttr(a, "href"))
		if err != nil {
			continue
		}
		arts[*u] = true
	}
	return arts, nil
}

func (disc *Discoverer) CookArticleURL(baseURL *url.URL, artLink string) (*url.URL, error) {
	// parse, extending to absolute
	u, err := baseURL.Parse(artLink)
	if err != nil {
		return nil, err
	}

	// on a host we accept?
	if !disc.isHostGood(u.Host) {
		return nil, fmt.Errorf("host rejected (%s)", u.Host)
	}

	// matches one of our url forms
	foo := u.RequestURI()
	accept := false
	for _, pat := range disc.ArtPats {
		if pat.MatchString(foo) {
			accept = true
			break
		}
	}
	if !accept {
		return nil, fmt.Errorf("url rejected")
	}

	// apply our sanitising rules for this site
	if disc.StripFragments {
		u.Fragment = ""
	}
	if disc.StripQuery {
		u.RawQuery = ""
	}
	return u, nil
}

func (disc *Discoverer) findNavLinks(root *html.Node) (LinkSet, error) {
	navLinks := make(LinkSet)
	if disc.NavLinkSel == nil {
		return navLinks, nil
	}
	for _, a := range disc.NavLinkSel.MatchAll(root) {
		link, err := disc.StartURL.Parse(GetAttr(a, "href"))
		if err != nil {
			continue
		}

		if !disc.isHostGood(link.Host) {
			continue
		}

		link.Fragment = ""

		navLinks[*link] = true
	}
	return navLinks, nil
}

// is host domain one we'll accept?
func (disc *Discoverer) isHostGood(host string) bool {
	if disc.HostPat != nil {
		return disc.HostPat.MatchString(host)
	}
	return host == disc.StartURL.Host
}

// GetAttr retrieved the value of an attribute on a node.
// Returns empty string if attribute doesn't exist.
func GetAttr(n *html.Node, attr string) string {
	for _, a := range n.Attr {
		if a.Key == attr {
			return a.Val
		}
	}
	return ""
}

// GetTextContent recursively fetches the text for a node
func GetTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	txt := ""
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		txt += GetTextContent(child)
	}

	return txt
}

// CompressSpace reduces all whitespace sequences (space, tabs, newlines etc) in a string to a single space.
// Leading/trailing space is trimmed.
// Has the effect of converting multiline strings to one line.
func CompressSpace(s string) string {
	multispacePat := regexp.MustCompile(`[\s]+`)
	s = strings.TrimSpace(multispacePat.ReplaceAllLiteralString(s, " "))
	return s
}
