package arts

// most of the public interface is defined here

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	//	"fmt"
	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type Author struct {
	Name    string `json:"name"`
	RelLink string
	Email   string
	Twitter string
}

type Publication struct {
	Name   string `json:"name,omitempty"`
	Domain string `json:"domain,omitempty"`

	// TODO: add publication versions of rel-author
	// eg "article:publisher", rel-publisher
}

type Article struct {
	CanonicalURL string `json:"canonical_url,omitempty"`
	// all known URLs for article (including canonical)
	// TODO: first url should be considered "preferred" if no canonical?
	URLs     []string `json:"urls,omitempty"`
	Headline string   `json:"headline,omitempty"`
	Authors  []Author `json:"authors,omitempty"`
	Content  string   `json:"content,omitempty"`
	// Published contains date of publication.
	// An ISO8601 string is used instead of time.Time, so that
	// less-precise representations can be held (eg YYYY-MM)
	Published   string      `json:"published,omitempty"`
	Updated     string      `json:"updated,omitempty"`
	Publication Publication `json:"publication,omitempty"`

	// TODO:
	// Language
	// Publication
	// article confidence?
}

// TODO:
// - detect non-article pages (index pages etc)

// TODO: pass hints in to the scraper:
// - is it contemporary? (ie not an insanely old or future date)
// - an expected author
// - expected location/timezone
// - expected language

// Debug is the global debug control for the scraper. Set up any loggers you want before calling Extract()
// By default all logging is suppressed.
var Debug = struct {
	// HeadlineLogger is where debug output from the headline extraction will be sent
	HeadlineLogger *log.Logger
	// AuthorsLogger is where debug output from the author extraction will be sent
	AuthorsLogger *log.Logger
	// ContentLogger is where debug output from the content extraction will be sent
	ContentLogger *log.Logger
	// DatesLogger is where debug output from the pubdate/lastupdated extraction will be sent
	DatesLogger *log.Logger
}{}

func Extract(client *http.Client, srcURL string) (*Article, error) {

	resp, err := client.Get(srcURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP error: %s", resp.Status))
	}

	rawHTML, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return ExtractHTML(rawHTML, srcURL)

}

func ExtractHTML(raw_html []byte, artUrl string) (*Article, error) {
	enc := findCharset("", raw_html)
	var r io.Reader
	r = strings.NewReader(string(raw_html))
	if enc != "utf-8" {
		// we'll be translating to utf-8
		var err error
		r, err = charset.NewReader(enc, r)
		if err != nil {
			return nil, err
		}
	}
	art := &Article{}

	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	// fill in any missing loggers to discard
	if Debug.HeadlineLogger == nil {
		Debug.HeadlineLogger = log.New(ioutil.Discard, "", 0)
	}
	if Debug.AuthorsLogger == nil {
		Debug.AuthorsLogger = log.New(ioutil.Discard, "", 0)
	}
	if Debug.ContentLogger == nil {
		Debug.ContentLogger = log.New(ioutil.Discard, "", 0)
	}
	if Debug.DatesLogger == nil {
		Debug.DatesLogger = log.New(ioutil.Discard, "", 0)
	}

	//	html.Render(dbug, root)

	removeScripts(root)
	// extract any canonical or alternate urls

	u, err := url.Parse(artUrl)
	if err != nil {
		return nil, err
	}

	art.CanonicalURL, art.URLs = grabURLs(root, u)
	if art.CanonicalURL != "" {
		artUrl = art.CanonicalURL
	}

	art.Publication = grabPublication(root, art)

	headline, headlineNode, err := grabHeadline(root, artUrl)
	if err == nil {
		art.Headline = headline
	}

	contentNodes, contentScores := grabContent(root)
	art.Authors = grabAuthors(root, contentNodes, headlineNode)

	published, updated := grabDates(root, artUrl, contentNodes)
	if !published.Empty() {
		art.Published = published.ISOFormat()
	}
	if !updated.Empty() {
		art.Updated = updated.ISOFormat()
	}

	// TODO: Turn all double br's into p's? Kill <style> tags? (see prepDocument())
	removeCruft(contentNodes, contentScores)
	sanitiseContent(contentNodes)

	var out bytes.Buffer
	for _, node := range contentNodes {
		html.Render(&out, node)
		out.WriteString("\n")
	}
	art.Content = out.String()
	// cheesyness to make it a little more readable...
	art.Content = regexp.MustCompile("(</p>)|(</div>)|(<br/>)").ReplaceAllString(art.Content, "$0\n")

	//	fmt.Printf("extracted %d nodes:\n", len(contentNodes))
	//	for _, n := range contentNodes {
	//		dumpTree(n, 0)
	//	}
	return art, nil
}

func grabPublication(root *html.Node, art *Article) Publication {
	// TODO: check og:site_name and other metadata
	pub := Publication{}

	// get domain
	canonical := art.CanonicalURL
	if canonical == "" {
		// TODO: better fallback
		canonical = art.URLs[0]
	}

	u, err := url.Parse(canonical)
	if err == nil {
		pub.Domain = u.Host
	}
	return pub
}
