package arts

import (
	"code.google.com/p/cascadia"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"regexp"
	"strings"
)

// a list of allowed elements and their allowed attrs
// all missing elements or attrs should be stripped
var elementWhitelist = map[atom.Atom][]atom.Atom{
	// basing on list at  https://developer.mozilla.org/en-US/docs/Web/Guide/HTML/HTML5/HTML5_element_list

	//Sections
	atom.Section: {},
	// atom.Nav?
	atom.Article: {},
	atom.Aside:   {},
	atom.H1:      {},
	atom.H2:      {},
	atom.H3:      {},
	atom.H4:      {},
	atom.H5:      {},
	atom.H6:      {},
	atom.Header:  {}, // should disallow?
	atom.Footer:  {}, // should disallow?
	atom.Address: {},
	//atom.Main?

	// Grouping content
	atom.P:          {},
	atom.Hr:         {},
	atom.Pre:        {},
	atom.Blockquote: {},
	atom.Ol:         {},
	atom.Ul:         {},
	atom.Li:         {},
	atom.Dl:         {},
	atom.Dt:         {},
	atom.Dd:         {},
	atom.Figure:     {},
	atom.Figcaption: {},
	atom.Div:        {},

	// Text-level semantics
	atom.A:      {atom.Href},
	atom.Em:     {},
	atom.Font:   {},
	atom.Strong: {},
	atom.Small:  {},
	atom.S:      {},
	atom.Cite:   {},
	atom.Q:      {},
	atom.Dfn:    {},
	atom.Abbr:   {atom.Title},
	// atom.Data
	atom.Time: {atom.Datetime},
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
	atom.Br:   {},
	atom.Wbr:  {},

	// Edits
	atom.Ins: {},
	atom.Del: {},

	//Embedded content
	atom.Img: {atom.Src, atom.Alt},
	// atom.Video?
	// atom.Audio?
	// atom.Map?
	// atom.Area?
	// atom.Svg?
	// atom.Math?

	// Tabular data
	atom.Table:    {},
	atom.Caption:  {},
	atom.Colgroup: {},
	atom.Col:      {},
	atom.Tbody:    {},
	atom.Thead:    {},
	atom.Tfoot:    {},
	atom.Tr:       {},
	atom.Td:       {},
	atom.Th:       {},

	// Forms

	// Interactive elements

}

func filterAttrs(n *html.Node, fn func(*html.Attribute) bool) {
	var out = make([]html.Attribute, 0)
	for _, a := range n.Attr {
		if fn(&a) {
			out = append(out, a)
		}
	}
	n.Attr = out
}

// Tidy up extracted content into something that'll produce reasonable html when
// rendered
// - remove comments
// - trim empty text nodes
// - TODO make links absolute
func tidyNode(node *html.Node) {
	var commentSel cascadia.Selector = func(n *html.Node) bool {
		return n.Type == html.CommentNode
	}
	var textSel cascadia.Selector = func(n *html.Node) bool {
		return n.Type == html.TextNode
	}
	var elementSel cascadia.Selector = func(n *html.Node) bool {
		return n.Type == html.ElementNode
	}

	// remove all comments
	for _, n := range commentSel.MatchAll(node) {
		n.Parent.RemoveChild(n)
	}

	leadingSpace := regexp.MustCompile(`^\s+`)
	trailingSpace := regexp.MustCompile(`\s+$`)
	// trim excessive leading/trailing space in text nodes, and cull empty ones
	for _, n := range textSel.MatchAll(node) {
		txt := leadingSpace.ReplaceAllStringFunc(n.Data, func(in string) string {
			if strings.Contains(in, "\n") {
				return "\n"
			} else {
				return " "
			}
		})
		txt = trailingSpace.ReplaceAllStringFunc(n.Data, func(in string) string {
			if strings.Contains(in, "\n") {
				return "\n"
			} else {
				return " "
			}
		})
		if len(strings.TrimSpace(txt)) == 0 {
			n.Parent.RemoveChild(n)
		} else {
			n.Data = txt
		}
	}

	// remove any elements or attrs not on the whitelist
	for _, n := range elementSel.MatchAll(node) {
		allowedAttrs, whiteListed := elementWhitelist[n.DataAtom]
		if !whiteListed {
			n.Parent.RemoveChild(n)
			continue
		}
		filterAttrs(n, func(attr *html.Attribute) bool {
			for _, allowed := range allowedAttrs {
				if attr.Key == allowed.String() {
					return true
				}
			}
			return false
		})
	}
}
