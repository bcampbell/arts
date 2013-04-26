package arts

// TODO:
// Url extraction:
// rel-canonical
// rel-shortlink
// og:whatever



import (
	"code.google.com/p/go.net/html"
	"strings"
	"bytes"
	"regexp"
	"os"
	"io"
	"io/ioutil"
)


type Author struct {
	Name string
	RelLink string	// rel-author link (or similar)
	Email string
	Twitter string
}

type Article struct {
	CanonicalUrl string
	Headline string
	Authors []Author
	Content string
	// Pubdate
	// Language
	// Publication
	// other URLs
}


func Extract(raw_html, artUrl string, debugOutput bool) (*Article, error) {
	r := strings.NewReader(raw_html)
	root, err := html.Parse(r)
	if err != nil {
		return nil,err
	}

	art := &Article{}

	var dbug io.Writer
	if debugOutput {
		dbug = os.Stderr
	} else {
		dbug = ioutil.Discard
	}

	removeScripts(root)
	contentNodes,contentScores := grabContent(root,dbug)
	art.Headline = grabHeadline(root, artUrl,dbug)
	art.Authors = grabAuthors(root, contentNodes, dbug)
	// TODO: Turn all double br's into p's? Kill <style> tags? (see prepDocument())
	removeCruft(contentNodes, contentScores)
	sanitiseContent(contentNodes)

	var out bytes.Buffer
	for _,node := range contentNodes {
		html.Render(&out,node)
		out.WriteString("\n")
	}
	art.Content = out.String()
	// cheesyness to make it a little more readable...
	art.Content = regexp.MustCompile("(</p>)|(</div>)|(<br/>)").ReplaceAllString(art.Content,"$0\n")

	art.CanonicalUrl = artUrl	// TODO!

//	fmt.Printf("extracted %d nodes:\n", len(contentNodes))
//	for _, n := range contentNodes {
//		dumpTree(n, 0)
//	}
	return art,nil
}


