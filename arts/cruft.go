package arts

// code to identify blocks of cruft
// - adverts
// - sidebars
// - related-articles
// - social media share buttons

import (
	//	"fmt"
	"code.google.com/p/cascadia"
	"golang.org/x/net/html"
	//	"golang.org/x/net/html/atom"
	"log"
	"regexp"
	"strings"
)

var cruftPats = struct {
	shareContainerSel       cascadia.Selector
	linkSel                 cascadia.Selector
	likelyShareContainerPat *regexp.Regexp
	likelyShareItemPat      *regexp.Regexp
	cruftIndicative         *regexp.Regexp
	shareLinkIndicative     []string
}{
	cascadia.MustCompile("ul,div"),
	cascadia.MustCompile("a"),
	regexp.MustCompile(`(?i)social|share|sharing|sharetools`),
	regexp.MustCompile(`(?i)twitter|google|gplus|googleplus|facebook|linkedin|whatsapp`),
	regexp.MustCompile(`(?i)\b(?:combx|comment|community|departments|disqus|livefyre|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|promo|rhs|sidebar|sponsor|shopping|tweet|twitter|facebook|trending)\b`),
	[]string{"plus.google.com", "facebook.com", "twitter.com", "pinterest.com", "linkedin.com", "mailto:", "whatsapp:"},
}

func findCruft(root *html.Node, dbug *log.Logger) []*html.Node {
	candidates := candidateList{}
	// look for likely ul or div blocks
	for _, el := range cruftPats.shareContainerSel.MatchAll(root) {
		elClass := getAttr(el, "class")
		elID := getAttr(el, "id")
		pat := cruftPats.cruftIndicative
		if pat.MatchString(elClass) || pat.MatchString(elID) {
			c := newStandardCandidate(el, "")
			c.addPoints(3, "cruft indicative")
			candidates = append(candidates, c)
		}
	}

	candidates.Sort()

	dbug.Printf("cruft blocks: %d candidates\n", len(candidates))
	for _, c := range candidates {
		c.dump(dbug)
	}
	cruft := []*html.Node{}
	for _, c := range candidates {
		cruft = append(cruft, c.node())
	}

	social := findSocialMediaShareBlocks(root, dbug)
	dbug.Printf("social blocks: %d candidates\n", len(social))
	for _, c := range social {
		c.dump(dbug)
	}

	for _, c := range social {
		cruft = append(cruft, c.node())
	}
	return cruft
}

func findSocialMediaShareBlocks(root *html.Node, dbug *log.Logger) candidateList {

	candidates := candidateList{}
	// look for likely containers
	for _, el := range cruftPats.shareContainerSel.MatchAll(root) {
		cls := getAttr(el, "class")
		id := getAttr(el, "id")
		if cruftPats.likelyShareContainerPat.MatchString(cls) ||
			cruftPats.likelyShareContainerPat.MatchString(id) {
			c := newStandardCandidate(el, "")
			c.addPoints(1, "likely share container")
			candidates = append(candidates, c)
		}
	}

	for _, c := range candidates {
		// TEST: contains likely links?
		for _, a := range cruftPats.linkSel.MatchAll(c.node()) {
			href := strings.ToLower(getAttr(a, "href"))
			for _, frag := range cruftPats.shareLinkIndicative {
				if strings.Contains(href, frag) {
					c.addPoints(2, "contains share link")
					continue
				}
			}
		}
	}

	//cull the duds
	candidates = candidates.Filter(func(c candidate) bool {
		return c.total() >= 4
	})

	// remove outermost container if nested
	candidates = candidates.Filter(func(c candidate) bool {
		for _, c2 := range candidates {
			if contains(c.node(), c2.node()) {
				return false
			}
		}
		return true
	})
	candidates.Sort()
	return candidates
}
