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
)

var cruftPats = struct {
	shareContainerSel       cascadia.Selector
	shareItemSel            cascadia.Selector
	likelyShareContainerPat *regexp.Regexp
	likelyShareItemPat      *regexp.Regexp
	cruftIndicative         *regexp.Regexp
}{
	cascadia.MustCompile("ul,div"),
	cascadia.MustCompile("li"),
	regexp.MustCompile(`(?i)social|share|sharetools`),
	regexp.MustCompile(`(?i)twitter|google|gplus|googleplus|facebook|linkedin|whatsapp`),
	regexp.MustCompile(`(?i)\b(?:combx|comment|community|disqus|livefyre|menu|remark|rss|shoutbox|side|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|promo|sponsor|shopping|tweet|twitter|facebook)\b`),
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
	for _, el := range social {
		cruft = append(cruft, el)
	}
	return cruft
}

func findSocialMediaShareBlocks(root *html.Node, dbug *log.Logger) []*html.Node {

	candidates := candidateList{}
	// look for likely ul or div blocks
	for _, el := range cruftPats.shareContainerSel.MatchAll(root) {
		cls := getAttr(el, "class")
		if cruftPats.likelyShareContainerPat.MatchString(cls) {
			c := newStandardCandidate(el, "")
			c.addPoints(1, "likely share container")
			candidates = append(candidates, c)
		}
	}

	for _, c := range candidates {
		// TEST: contains likely links?
		for _, el := range cruftPats.shareItemSel.MatchAll(c.node()) {
			cls := getAttr(el, "class")
			if cruftPats.likelyShareItemPat.MatchString(cls) {
				c.addPoints(1, "likely share item")
			}
		}
	}

	culled := candidateList{}
	for _, c := range candidates {

		keep := true
		// remove outermost container if nested
		for _, c2 := range candidates {
			if contains(c.node(), c2.node()) {
				keep = false
			}
		}
		//
		if c.total() < 4 {
			keep = false
		}

		if keep {
			culled = append(culled, c)
		}
	}
	candidates = culled
	candidates.Sort()

	dbug.Printf("Social blocks: %d candidates\n", len(candidates))
	for _, c := range candidates {
		c.dump(dbug)
	}

	socialBlocks := []*html.Node{}
	for _, c := range candidates {
		socialBlocks = append(socialBlocks, c.node())
	}

	return socialBlocks
}
