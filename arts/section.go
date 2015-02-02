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
	meta cascadia.Selector
}{
	cascadia.MustCompile(`head meta[property="article:section"], head meta[name="Section"]`),
}

// evil special-case hacks for various specific sites
var evilSectionHacks = struct {
	script       cascadia.Selector
	ftJSPat      *regexp.Regexp
	skyNewsJSPat *regexp.Regexp
	itvSel       cascadia.Selector
}{
	cascadia.MustCompile(`script`),
	// siteMapTerm = 'Sections.Technology';
	regexp.MustCompile(`siteMapTerm = '(?:.*)[.](.*)';`),
	// window.skynews.config.analytics.section = 'politics/Leaders Await Grilling On Issues Facing Young';
	regexp.MustCompile(`analytics[.]section = '(.*)/.*';`),

	// <li class="tag-list__tag tag-list__tag--category"><a href="/news/health/">Health</a></li>
	cascadia.MustCompile(`.tag-list__tag--category a`),
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
	if section == "" {
		// no section found - try out assorted evil special-case per-site hacks
		switch strings.ToLower(u.Host) {
		case "www.ft.com":
			return sectionFromJavascript(root, evilSectionHacks.ftJSPat)
		case "news.sky.com":
			return sectionFromJavascript(root, evilSectionHacks.skyNewsJSPat)
		case "www.itv.com":
			return sectionFromElement(root, evilSectionHacks.itvSel)
		}
	}

	return section
}

// Evil little special-case to get an article section out javascript.
// Sites often embed metadata in javascript which isn't marked up any other way.
// Usually, this is extra data is used for advertising. Sigh.
func sectionFromJavascript(root *html.Node, pat *regexp.Regexp) string {
	for _, el := range evilSectionHacks.script.MatchAll(root) {
		txt := getTextContent(el)
		m := pat.FindStringSubmatch(txt)
		if m != nil {
			return strings.TrimSpace(strings.ToLower(m[1]))
		}
	}
	return ""
}

// grab text of matching selector, return as one line
func sectionFromElement(root *html.Node, sel cascadia.Selector) string {
	el := sel.MatchFirst(root)
	if el == nil {
		return ""
	}
	return compressSpace(getTextContent(el))
}
