package arts

// most of the public interface is defined here

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"io"
	"os"
	"regexp"
	"strings"
	//	"fmt"
	"code.google.com/p/go-charset/charset"
	_ "code.google.com/p/go-charset/data"
	"io/ioutil"
)

type Author struct {
	Name    string
	RelLink string // rel-author link (or similar)
	Email   string
	Twitter string
}

type Article struct {
	CanonicalUrl  string
	AlternateUrls []string // doesn't include canonical one
	Headline      string
	Authors       []Author
	Content       string
	// date of publication (an ISO8601 string or "" for none)
	Published string
	// Updated
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

func Extract(raw_html []byte, artUrl string, debugOutput bool) (*Article, error) {
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

	var dbug io.Writer
	if debugOutput {
		dbug = os.Stderr
	} else {
		dbug = ioutil.Discard
	}

	removeScripts(root)
	// extract any canonical or alternate urls
	art.CanonicalUrl, art.AlternateUrls = grabUrls(root)

	contentNodes, contentScores := grabContent(root, dbug)
	art.Headline = grabHeadline(root, artUrl, dbug)
	art.Authors = grabAuthors(root, contentNodes, dbug)

	art.Published, _ = grabDates(root, art.CanonicalUrl, dbug)

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
