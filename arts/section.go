package arts

// section.go - code to extract the section an article appears in
// ie Politics/Sport/whatever...

import (
	"code.google.com/p/cascadia"
	"golang.org/x/net/html"
	"net/url"
	"regexp"
	"strings"
)

// eg <meta property="article:section" content="Politics" />
var sectionSels = struct {
	meta      cascadia.Selector
	script    cascadia.Selector
	ftSectPat *regexp.Regexp
}{
	cascadia.MustCompile(`head meta[property="article:section"], head meta[name="Section"]`),
	cascadia.MustCompile(`script`),
	regexp.MustCompile(`siteMapTerm = '(?:.*)[.](.*)';`),
}

// returns "" if no section found
// if multiple sections, return as comma-separated list
func grabSection(root *html.Node, u *url.URL) string {
	raw := map[string]struct{}{}

	for _, el := range sectionSels.meta.MatchAll(root) {
		foo := getAttr(el, "content")
		foo = strings.ToLower(strings.TrimSpace(foo))
		if foo != "" {
			raw[foo] = struct{}{}
		}
	}

	out := make([]string, len(raw))
	i := 0
	for txt, _ := range raw {
		out[i] = txt
		i++
	}
	section := strings.Join(out, ", ")
	if section == "" && strings.ToLower(u.Host) == "www.ft.com" {
		return ftSection(root)
	}

	return section
}

// evil little special-case to get a section out of an FT article...
// look for javascript, eg:
// siteMapTerm = 'Sections.Technology';
func ftSection(root *html.Node) string {
	for _, el := range sectionSels.script.MatchAll(root) {
		txt := getTextContent(el)
		m := sectionSels.ftSectPat.FindStringSubmatch(txt)
		if m != nil {
			return strings.TrimSpace(strings.ToLower(m[1]))
		}
	}
	return ""
}
