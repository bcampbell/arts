package arts

import (
	"code.google.com/p/go.net/html"
	"strings"
	"bytes"
	"regexp"
)


type Author struct {
	Name string
	URL string
	Email string
	Twitter string
}

type Article struct {
	Headline string
	Authors []Author
	Content string
	// Pubdate
	// Language
	// Publication
}


func Extract(raw_html, artUrl string) (*Article, error) {
	r := strings.NewReader(raw_html)
	root, err := html.Parse(r)
	if err != nil {
		return nil,err
	}

	art := &Article{}

	removeScripts(root)
	contentNodes,contentScores := grabContent(root)
	art.Headline = grabHeadline(root, artUrl)
	art.Authors = grabAuthors(root, contentNodes)
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


//	fmt.Printf("extracted %d nodes:\n", len(contentNodes))
//	for _, n := range contentNodes {
//		dumpTree(n, 0)
//	}
	return art,nil
}


