package arts

// util.go holds generally useful stuff, mainly to do with using html.Nodes.
// Some of this might justify a separate package...

import (
	"fmt"
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/text/unicode/norm"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// wrapper for reversing any sortable
type Reverse struct {
	sort.Interface
}

func (r Reverse) Less(i, j int) bool {
	return r.Interface.Less(j, i)
}

// compressSpace reduces all whitespace sequences (space, tabs, newlines etc) in a string to a single space.
// Leading/trailing space is trimmed.
// Has the effect of converting multiline strings to one line.
func compressSpace(s string) string {
	multispacePat := regexp.MustCompile(`[\s]+`)
	s = strings.TrimSpace(multispacePat.ReplaceAllLiteralString(s, " "))
	return s
}

// prep text for comparisons, in a language-neutral way
func normaliseText(txt string) string {
	txt = norm.NFKD.String(txt)
	txt = strings.ToLower(txt)
	txt = compressSpace(txt)
	return txt
}

// toAlphanumeric converts a utf8 string into plain ascii, alphanumeric only.
// It tries to replace latin-alphabet accented characters with the plain-ascii bases.
// NOTE: will return empty string for non-latin text
func toAlphanumeric(txt string) string {
	// convert to NFKD form
	// eg, from wikipedia:
	// "U+00C5" (the Swedish letter "Å") is expanded into "U+0041 U+030A" (Latin letter "A" and combining ring above "°")
	n := norm.NFKD.String(txt)

	// strip out non-ascii chars (eg combining ring above "°", leaving just "A")
	n = strings.Map(
		func(r rune) rune {
			if r > 128 {
				r = -1
			}
			return r
		}, n)

	n = regexp.MustCompile(`[^a-zA-Z0-9 ]`).ReplaceAllLiteralString(n, "")
	n = compressSpace(n)
	n = strings.ToLower(n)
	return n
}

// getSlug extracts the slug part of a url, if present (else returns "")
func getSlug(rawurl string) string {

	o, err := url.Parse(rawurl)
	if err != nil {
		return ""
	}
	slugpat := regexp.MustCompile(`(?i)((?:[a-z0-9]+[-_])+(?:[a-z0-9]+?))(?:[.][a-z0-9]{3,5})?$`)
	m := slugpat.FindStringSubmatch(o.Path)
	if m == nil {
		return ""
	}

	return m[1]
}

// walkChildren iterates over all the descendants of root in top-down order,
// invoking fn upon each one
func walkChildren(root *html.Node, fn func(*html.Node)) {
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		fn(child)
		walkChildren(child, fn)
	}
}

//
func closest(n *html.Node, sel cascadia.Selector) *html.Node {
	for n != nil {
		if sel.Match(n) {
			break
		}
		n = n.Parent
	}
	return n
}

// return a slice containing all the parents of this node up to root
func parentNodes(n *html.Node) []*html.Node {
	foo := make([]*html.Node, 0)
	for n.Parent != nil {
		n = n.Parent
		foo = append(foo, n)
	}
	return foo
}

// contains returns true if is a descendant of container
func contains(container *html.Node, n *html.Node) bool {
	n = n.Parent
	for ; n != nil; n = n.Parent {
		if n == container {
			return true
		}
	}
	return false
}

// getAttr retrieved the value of an attribute on a node.
// Returns empty string if attribute doesn't exist.
func getAttr(n *html.Node, attr string) string {
	for _, a := range n.Attr {
		if a.Key == attr {
			return a.Val
		}
	}
	return ""
}

// getTextContent recursively fetches the text for a node
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

// getLinkDensity calculates the ratio of link text to overall text in a node.
// 0 means no link text, 1 means everything is link text
func getLinkDensity(n *html.Node) float64 {
	textLength := len(compressSpace(getTextContent(n)))
	linkLength := 0
	linkSel := cascadia.MustCompile("a")
	for _, a := range linkSel.MatchAll(n) {
		linkLength += len(compressSpace(getTextContent(a)))
	}

	return float64(linkLength) / float64(textLength)
}

// describeNode generates a debug string describing the node.
// returns a string of form: "<element#id.class>" (ie, like a css selector)
func describeNode(n *html.Node) string {
	switch n.Type {
	case html.ElementNode:
		desc := n.DataAtom.String()
		if n.DataAtom == atom.Meta {
			for _, attrName := range []string{"name", "property"} {
				attrVal := getAttr(n, attrName)
				if attrVal != "" {
					desc = fmt.Sprintf("%s[%s=\"%s\"]", n.DataAtom.String(), attrName, attrVal)
					break
				}
			}
		} else {
			id := getAttr(n, "id")
			if id != "" {
				desc = desc + "#" + id
			}
			// TODO: handle multiple classes (eg "h1.heading.fancy")
			cls := getAttr(n, "class")
			if cls != "" {
				desc = desc + "." + cls
			}
		}
		return "<" + desc + ">"
	case html.TextNode:
		return fmt.Sprintf("{TextNode} %s", strconv.Quote(n.Data))
	case html.DocumentNode:
		return "{DocumentNode}"
	case html.CommentNode:
		return "{Comment}"
	case html.DoctypeNode:
		return "{DoctypeNode}"
	}
	return "???" // not an element
}

// dumpTree is a debug helper to display a tree of nodes
func dumpTree(n *html.Node, depth int) {
	fmt.Printf("%s%s\n", strings.Repeat(" ", depth), describeNode(n))
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		dumpTree(child, depth+1)
	}
}

// prevNode returns the previous node (ie walks backward in document order)
func prevNode(n *html.Node) *html.Node {
	if n.PrevSibling != nil {
		for n = n.PrevSibling; n.LastChild != nil; n = n.LastChild {
		}
		return n
	}
	return n.Parent
}

func wordCount(s string) int {
	return len(strings.Fields(s))
}

// jaccardWordCompare compares strings based on the words they contain
// returns a value between 0 (no match) and 1 (perfect match)
// Calculates the Jaccard index on the sets of words each string contains.
// https://en.wikipedia.org/wiki/Jaccard_index
func jaccardWordCompare(a string, b string) float64 {
	aWords := strings.Fields(a)
	bWords := strings.Fields(b)
	lookup := make(map[string]bool)
	for _, word := range aWords {
		lookup[word] = true
	}

	var intersectCnt float64 = 0
	for _, word := range bWords {
		if _, exists := lookup[word]; exists {
			intersectCnt++
		}
	}

	// now add the rest of the words to the lookup to calculate the union
	for _, word := range bWords {
		lookup[word] = true
	}
	unionCnt := float64(len(lookup))
	if unionCnt == 0 {
		return 1 // both a and b empty!
	}
	return intersectCnt / unionCnt
}

func nextNode(n *html.Node) *html.Node {
	if n.FirstChild != nil {
		return n.FirstChild
	}
	if n.NextSibling != nil {
		return n.NextSibling
	}
	for {
		n = n.Parent
		if n == nil {
			return nil
		}
		if n.NextSibling != nil {
			return n.NextSibling
		}

	}
	return nil
}

func nextElement(e *html.Node) *html.Node {
	for {
		e = nextNode(e)
		if e == nil {
			return nil
		}
		if e.Type == html.ElementNode {
			return e
		}
	}
}

func interveningElements(e1, e2 *html.Node) ([]*html.Node, error) {
	out := []*html.Node{}
	n := e1
	for {
		n = nextElement(n)
		if n == e2 {
			break
		}
		if n == nil {
			return nil, fmt.Errorf("e2 not found")
		}

		out = append(out, n)
	}
	return out, nil
}

// return a snippet of text, up to n chars
// adds "..." if truncated
func snip(s string, n int) string {

	// TODO: fix single-byte rune assumption
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// from github.com/bcampbell/htmlutil
var inlineNodes = map[atom.Atom]struct{}{
	atom.A:      {},
	atom.Em:     {},
	atom.Strong: {},
	atom.Small:  {},
	atom.S:      {},
	atom.Cite:   {},
	atom.Q:      {},
	atom.Dfn:    {},
	atom.Abbr:   {},
	// atom.Data
	atom.Time: {},
	atom.Code: {},
	atom.Var:  {},
	atom.Samp: {},
	atom.Kbd:  {},
	atom.Sub:  {},
	atom.Sup:  {},
	atom.I:    {},
	atom.B:    {},
	atom.U:    {},
	atom.Mark: {},
	atom.Ruby: {},
	atom.Rt:   {},
	atom.Rp:   {},
	atom.Bdi:  {},
	atom.Bdo:  {},
	atom.Span: {},
	//      atom.Br:   {},
	atom.Wbr: {},
	atom.Ins: {},
	atom.Del: {},
}

func containsBlockElements(n *html.Node) bool {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		_, inline := inlineNodes[child.DataAtom]
		if !inline {
			return true
		}
		if containsBlockElements(child) {
			return true
		}
	}

	return false
}
