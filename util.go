package main

import (
	"fmt"
    "bytes"
    "net/url"
    "regexp"
    "strings"
    "code.google.com/p/go.net/html"
    "code.google.com/p/go.text/unicode/norm"
	"github.com/matrixik/goquery"
	"code.google.com/p/cascadia"
)

// writeNodeText writes the text contained in n and its descendants to b.
func writeNodeText(n *html.Node, b *bytes.Buffer) {
	switch n.Type {
	case html.TextNode:
		b.WriteString(n.Data)
	case html.ElementNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeNodeText(c, b)
		}
	}
}

// nodeText returns the text contained in n and its descendants.
func nodeText(n *html.Node) string {
	var b bytes.Buffer
	writeNodeText(n, &b)
	return b.String()
}

// nodeOwnText returns the contents of the text nodes that are direct
// children of n.
func nodeOwnText(n *html.Node) string {
	var b bytes.Buffer
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return b.String()
}


// compress all whitespace sequences (space, tabs, newlines etc) to single space
// leading/trailing space is trimmed
// also has the effect of converting multiline strings to oneliners
func compressSpace(s string) string {
    multispacePat := regexp.MustCompile(`[\s]+`)
    s = strings.TrimSpace(multispacePat.ReplaceAllLiteralString(s," "))
    return s
}


func toAlphanumeric(txt string) string {
    // convert to NFKD form
    // eg, from wikipedia:
    // "U+00C5" (the Swedish letter "Å") is expanded into "U+0041 U+030A" (Latin letter "A" and combining ring above "°")
    n := norm.NFKD.String(txt)

    // strip out non-ascii chars (eg combining ring above "°", leaving just "A")
    n = strings.Map(
        func(r rune) rune {
            if r>128 {
                r = -1
            }
            return r
        }, n)

    n = regexp.MustCompile(`[^a-zA-Z0-9 ]`).ReplaceAllLiteralString(n,"")
    n = compressSpace(n)
    n = strings.ToLower(n)
    return n
}

func getSlug(rawurl string) string {

    o,err := url.Parse(rawurl)
    if err != nil {
        return ""
    }
    slugpat := regexp.MustCompile(`((?:[a-zA-Z0-9]+[-_])+[a-zA-Z0-9]+)`)
    m := slugpat.FindStringSubmatch(o.Path)
    if m==nil {
        return ""
    }

    return m[0]
}



func insideArticle(s *goquery.Selection) bool {
	if s.Closest("article, #post, .article, .story-body").Length()>0 {
		return true
	}


	return false
}


// get the value of the attribute, or return "" if not exists
func getAttr(n *html.Node, attr string) string {
	for _, a := range n.Attr {
		if a.Key == attr {
			return a.Val
		}
	}
	return ""
}

func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	txt := ""
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		txt += getTextContent(child)
	}

	return txt
}

// return a string of form: "element#id.class" (ie, in css selector form)
func describeNode(n *html.Node) string {
	switch n.Type {
	case html.ElementNode:
		desc := n.DataAtom.String()
		id := getAttr(n,"id")
		if id != "" {
			desc = desc + "#" + id
		}
		// TODO: handle multiple classes (eg "h1.heading.fancy")
		cls := getAttr(n,"class")
		if cls != "" {
			desc = desc + "." + cls
		}
		return "<" + desc + ">"
	case html.TextNode:
		return "{TextNode}"
	case html.DocumentNode:
		return "{DocumentNode}"
	case html.CommentNode:
		return "{Comment}"
	case html.DoctypeNode:
		return "{DoctypeNode}"
	}
	return "???"	// not an element
}

// return the link density of the node content, in range [0..1]
func getLinkDensity(n *html.Node) float64 {
	textLength := len(getTextContent(n))
	linkLength := 0
	linkSel := cascadia.MustCompile("a")
	for _,a := range(linkSel.MatchAll(n)) {
		linkLength += len(getTextContent(a))
	}

	return float64(linkLength) / float64(textLength)
}

// helper for debugging
func dumpNode(n *html.Node, depth int) {
	fmt.Printf("%s%s\n", strings.Repeat(" ", depth), describeNode(n))
	for child:=n.FirstChild; child != nil; child=child.NextSibling {
		dumpNode(child,depth+1)
	}
}

