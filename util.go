package main

import (
    "bytes"
    "net/url"
    "regexp"
    "strings"
    "code.google.com/p/go.net/html"
    "code.google.com/p/go.text/unicode/norm"
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

