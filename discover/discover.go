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
	"code.google.com/p/go.net/html"
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

type Discoverer struct {
	Name           string
	StartURL       url.URL
	ArtPats        []*regexp.Regexp
	NavLinkSel     cascadia.Selector
	StripFragments bool
	StripQuery     bool

	ErrorLog Logger
	InfoLog  Logger
	Stats    struct {
		ErrorCount int
		FetchCount int
	}
}

func (disc *Discoverer) Run(client *http.Client) (LinkSet, error) {

	queued := make(LinkSet) // nav pages to scan for article links
	seen := make(LinkSet)   // nav pages we've scanned
	arts := make(LinkSet)   // article links we've found so far

	queued.Add(disc.StartURL)

	for len(queued) > 0 {
		u := queued.Pop()
		seen.Add(u)
		//

		root, err := disc.fetchAndParse(client, &u)
		if err != nil {
			disc.ErrorLog.Printf("%s\n", err.Error())
			disc.Stats.ErrorCount++
			if disc.Stats.ErrorCount > disc.Stats.FetchCount/10 {
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

		foo, err := disc.findArticles(root)
		if err != nil {
			return nil, err
		}
		arts.Merge(foo)

		disc.InfoLog.Printf("Visited %s, found %d articles\n", u.String(), len(foo))
	}

	return arts, nil
}

func (disc *Discoverer) fetchAndParse(client *http.Client, pageURL *url.URL) (*html.Node, error) {
	resp, err := http.Get(pageURL.String())
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

func (disc *Discoverer) findArticles(root *html.Node) (LinkSet, error) {
	arts := make(LinkSet)
	for _, a := range aSel.MatchAll(root) {
		// fetch url and extend to absolute
		link, err := disc.StartURL.Parse(GetAttr(a, "href"))
		if err != nil {
			continue
		}

		if link.Host != disc.StartURL.Host {
			continue
		}

		foo := link.RequestURI()
		accept := false
		for _, pat := range disc.ArtPats {
			if pat.MatchString(foo) {
				accept = true
				break
			}
		}
		if !accept {
			continue
		}

		if disc.StripFragments {
			link.Fragment = ""
		}
		if disc.StripQuery {
			link.RawQuery = ""
		}

		arts[*link] = true
	}
	return arts, nil
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
		if link.Host != disc.StartURL.Host {
			continue
		}

		link.Fragment = ""

		navLinks[*link] = true
	}
	return navLinks, nil
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
