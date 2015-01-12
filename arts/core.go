package arts

// most of the public interface is defined here

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/net/html"
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

type Keyword struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
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
	Keywords    []Keyword   `json:"keywords,omitempty"`
	Section     string      `json:"section,omitempty"`
	// TODO:
	// Language
	// article confidence?
}

func (art *Article) BestURL() string {
	if art.CanonicalURL != "" {
		return art.CanonicalURL
	}
	if len(art.URLs) > 0 {
		return art.URLs[0]
	}
	return ""
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

	// URLLoggger is where debug output from URL extraction will be sent (rel-canonical etc)
	URLLogger *log.Logger
}{}

// delete this and leave it up to user?
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

	return ExtractFromHTML(rawHTML, srcURL)

}

func ParseHTML(rawHTML []byte) (*html.Node, error) {
	enc := findCharset("", rawHTML)
	var r io.Reader
	r = strings.NewReader(string(rawHTML))
	if enc != "utf-8" {
		// we'll be translating to utf-8
		var err error
		r, err = charset.NewReader(enc, r)
		if err != nil {
			return nil, err
		}
	}

	return html.Parse(r)
}

func ExtractFromHTML(rawHTML []byte, artURL string) (*Article, error) {

	root, err := ParseHTML(rawHTML)
	if err != nil {
		return nil, err
	}

	return ExtractFromTree(root, artURL)
}

func ExtractFromTree(root *html.Node, artURL string) (*Article, error) {

	art := &Article{}

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
	if Debug.URLLogger == nil {
		Debug.URLLogger = log.New(ioutil.Discard, "", 0)
	}

	//	html.Render(dbug, root)
	u, err := url.Parse(artURL)
	if err != nil {
		return nil, err
	}

	art.Section = grabSection(root, u)

	removeScripts(root)
	// extract any canonical or alternate urls

	art.CanonicalURL, art.URLs = grabURLs(root, u)
	if art.CanonicalURL != "" {
		artURL = art.CanonicalURL
	}

	art.Publication = grabPublication(root, art)
	art.Keywords = grabKeywords(root)

	headline, headlineNode, err := grabHeadline(root, artURL)
	if err == nil {
		art.Headline = headline
	}

	contentNodes, contentScores := grabContent(root)
	art.Authors = grabAuthors(root, contentNodes, headlineNode)

	published, updated := grabDates(root, artURL, contentNodes, headlineNode)
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
