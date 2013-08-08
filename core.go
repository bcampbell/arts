package arts

// most of the public interface is defined here

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"io"
	"regexp"
	"strings"
	//	"fmt"
	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
	"io/ioutil"
	"log"
	"net/url"
)

type Author struct {
	Name    string
	RelLink string // rel-author link (or similar)
	Email   string
	Twitter string
}

type Article struct {
	CanonicalUrl string
	Urls         []string // all known URLs for article
	Headline     string
	Authors      []Author
	Content      string
	// date of publication (an ISO8601 string or "" for none)
	Published string
	Updated   string
	// Language
	// Publication
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

func Extract(raw_html []byte, artUrl string) (*Article, error) {
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

	art.CanonicalUrl, art.Urls = grabUrls(root, u)

	headline, headlineNode, err := grabHeadline(root, artUrl)
	if err == nil {
		art.Headline = headline
	}

	contentNodes, contentScores := grabContent(root)
	art.Authors = grabAuthors(root, contentNodes, headlineNode)
	published, updated := grabDates(root, art.CanonicalUrl, contentNodes)
	if !published.Empty() {
		art.Published, _ = published.IsoFormat()
	}
	if !updated.Empty() {
		art.Updated, _ = updated.IsoFormat()
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
