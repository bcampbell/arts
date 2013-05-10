package arts

import (
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"fmt"
	//"github.com/matrixik/goquery"
	"code.google.com/p/cascadia"
	"io"
	"regexp"
	"sort"
	//	"strings"
	//	"errors"
	"github.com/bcampbell/fuzzytime"
	"strconv"
)

type dateCandidate struct {
	standardCandidate
	dt fuzzytime.DateTime
}

func newDateCandidate(n *html.Node, txt string, dt fuzzytime.DateTime) candidate {
	return &dateCandidate{standardCandidate{n, txt, 0, 1, make([]string, 0, 4)}, dt}
}

var dateSels = struct {
	machineReadable cascadia.Selector
	meta            cascadia.Selector
	tags            cascadia.Selector
	hatomPublished  cascadia.Selector
	hatomUpdated    cascadia.Selector
}{
	cascadia.MustCompile(`time, .published, .updated`),
	cascadia.MustCompile(`head meta`),
	//cascadia.MustCompile(`time,p,span,div,li,td,th,h4,h5,h6,font`),
	cascadia.MustCompile(`span`),
	cascadia.MustCompile(`hentry .published`),
	cascadia.MustCompile(`hentry .updated`),
}

var datePats = struct {
	publishedIndicativeText *regexp.Regexp
	updatedIndicativeText   *regexp.Regexp
	urlDateFmts             []*regexp.Regexp // to get dates out of URLs
	genericClasses          *regexp.Regexp
	publishedClasses        *regexp.Regexp
	updatedClasses          *regexp.Regexp
}{
	// publishedIndicativeText
	regexp.MustCompile(`published|posted`),
	// updatedIndicativeText
	regexp.MustCompile(`updated|last modified`),
	// urlDateFmts
	[]*regexp.Regexp{
		regexp.MustCompile(`/(?P<year>\d{4})/(?P<month>\d{1,2})/(?P<day>\d{1,2})/`),
		regexp.MustCompile(`[^0-9](?P<year>\d{4})-(?P<month>\d{1,2})-(?P<day>\d{1,2})[^0-9]`),
		// TODO: should accept YYYY/MM with missing day?
	},
	// genericClasses
	regexp.MustCompile(`(?i)updated|date|time|fecha`),
	// publishedClasses
	regexp.MustCompile(`(?i)published`),
	// updatedClasses
	regexp.MustCompile(`(?i)modified|updated`),
}

// dateFromURl looks for an obvious date in the url
func dateFromUrl(url string) (d fuzzytime.DateTime) {

	for _, pat := range datePats.urlDateFmts {
		m := pat.FindStringSubmatch(url)
		if m != nil {
			year, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			month, err := strconv.Atoi(m[2])
			if err != nil {
				continue
			}
			day, err := strconv.Atoi(m[3])
			if err != nil {
				continue
			}
			d.SetYear(year)
			d.SetMonth(month)
			d.SetDay(day)
			break
		}
	}
	return
}

/* TODO: some more <meta> tags, from cnn.com:
   <meta name="pubdate" itemprop="datePublished" content="2013-05-07T10:45:16Z">
   <meta name="lastmod" itemprop="dateModified" content="2013-05-07T10:55:21Z">
   <meta itemprop="dateCreated" content="2013-05-07T10:45:16Z">
*/
// TODO: other meta tags?
// "DCSext.articleFirstPublished"
// "DC.date.issued"
// "last-modified"

// check meta tags for anything useful
// eg
// <meta property="article:published_time" content="2013-05-02" />
// <meta content="2013-05-05T11:30:09Z" property="article:modified_time">

// datesFromMeta checks for timestamps in <meta> tags.
// returns published, updated
func datesFromMeta(root *html.Node) (fuzzytime.DateTime, fuzzytime.DateTime) {
	metaPublished := fuzzytime.DateTime{}
	metaUpdated := fuzzytime.DateTime{}
	for _, node := range dateSels.meta.MatchAll(root) {
		prop := getAttr(node, "property")
		if prop == "article:published_time" {
			content := getAttr(node, "content")
			metaPublished = fuzzytime.Extract(content)
		} else if prop == "article:modified_time" {
			content := getAttr(node, "content")
			metaUpdated = fuzzytime.Extract(content)
		}
	}
	return metaPublished, metaUpdated
}

// machine readable times:
// express
// <time itemprop="datePublished" datetime="2013-05-05T21:35:22" class="published-date">

//
//
//
func grabDates(root *html.Node, url string, contentNodes []*html.Node, dbug io.Writer) (fuzzytime.DateTime, fuzzytime.DateTime) {
	var publishedCandidates = make(candidateList, 0, 32)
	var updatedCandidates = make(candidateList, 0, 32)

	// there might be an obvious date in the URL
	urlDate := dateFromUrl(url)

	// look for timestamps in <meta> tags
	metaPublished, metaUpdated := datesFromMeta(root)

	if metaPublished.HasFullDate() && metaUpdated.HasFullDate() {
		return metaPublished, metaUpdated
	}

	for _, node := range dateSels.tags.MatchAll(root) {

		var txt string
		// a couple of cases where we want text from attrs instead
		switch node.DataAtom {
		case atom.Time:
			txt = getAttr(node, "datetime")
			if txt == "" {
				txt = getTextContent(node)
			}
		case atom.Abbr:
			txt = getAttr(node, "title")
			if txt == "" {
				txt = getTextContent(node)
			}
		default:
			txt = getTextContent(node)
		}

		if len(txt) < 6 || len(txt) > 150 {
			continue // too short
		}

		// got some date/time info?
		dt := fuzzytime.Extract(txt)
		// TODO: ensure enough date info to be useful
		if dt.Empty() {
			continue // nope
		}

		publishedC := newDateCandidate(node, txt, dt)
		updatedC := newDateCandidate(node, txt, dt)

		// TEST: is machine readable?
		if node.DataAtom == atom.Time {
			publishedC.addPoints(1, "<time>")
			updatedC.addPoints(1, "<time>")
		}

		// TEST: indicative text ("posted:" etc...)
		if datePats.publishedIndicativeText.MatchString(txt) {
			publishedC.addPoints(1, "indicative text")
		}
		// TEST: indicative text ("posted:" etc...)
		if datePats.updatedIndicativeText.MatchString(txt) {
			updatedC.addPoints(1, "indicative text")
		}

		// TEST: hAtom date markup
		if dateSels.hatomPublished.Match(node) {
			publishedC.addPoints(2, "hentry .published")
		}
		if dateSels.hatomUpdated.Match(node) {
			publishedC.addPoints(2, "hentry .updated")
		}

		// TEST: likely class or id?
		if datePats.genericClasses.MatchString(getAttr(node, "class")) {
			updatedC.addPoints(1, "likely class")
			publishedC.addPoints(1, "likely class")
		}
		if datePats.genericClasses.MatchString(getAttr(node, "id")) {
			updatedC.addPoints(1, "likely id")
			publishedC.addPoints(1, "likely id")
		}
		// TEST: likely class or id for published?
		if datePats.publishedClasses.MatchString(getAttr(node, "class")) {
			publishedC.addPoints(1, "likely class for published")
		}
		if datePats.publishedClasses.MatchString(getAttr(node, "id")) {
			publishedC.addPoints(1, "likely id for published")
		}
		// TEST: likely class or id for updated?
		if datePats.updatedClasses.MatchString(getAttr(node, "class")) {
			updatedC.addPoints(1, "likely class for updated")
		}
		if datePats.updatedClasses.MatchString(getAttr(node, "id")) {
			updatedC.addPoints(1, "likely id for updated")
		}

		// TEST: within article content?
		for _, contentNode := range contentNodes {
			if contains(contentNode, node) {
				publishedC.addPoints(1, "contained within content")
				updatedC.addPoints(1, "contained within content")
			}
		}
		// TEST: share a parent with content?
		for _, contentNode := range contentNodes {
			if contains(contentNode.Parent, node) {
				publishedC.addPoints(1, "near content")
				updatedC.addPoints(1, "near content")
			}
		}

		// TEST: matches date info in URL?
		if !urlDate.Empty() {
			if urlDate.HasYear() && dt.HasYear() && urlDate.Year() == dt.Year() {
				updatedC.addPoints(1, "matches year in url")
				publishedC.addPoints(1, "matches year in url")
			}
			if urlDate.HasMonth() && dt.HasMonth() && urlDate.Month() == dt.Month() {
				updatedC.addPoints(1, "matches month in url")
				publishedC.addPoints(1, "matches month in url")
			}
			if urlDate.HasDay() && dt.HasDay() && urlDate.Day() == dt.Day() {
				updatedC.addPoints(1, "matches day in url")
				publishedC.addPoints(1, "matches day in url")
			}
		}

		// TODO: TEST: agrees with <meta> tag values?

		// TODO: TEST - proximity to top or bottom of article content
		// TODO: check for value-title pattern?
		if publishedC.total() > 0 {
			publishedCandidates = append(publishedCandidates, publishedC)
		}

		if updatedC.total() > 0 {
			updatedCandidates = append(updatedCandidates, updatedC)
		}

	}

	sort.Sort(Reverse{updatedCandidates})
	fmt.Fprintf(dbug, "meta updated: '%s\n", metaUpdated.String())
	fmt.Fprintf(dbug, "UPDATED: %d candidates\n", len(updatedCandidates))
	for _, c := range updatedCandidates {
		c.dump(dbug)
	}

	sort.Sort(Reverse{publishedCandidates})
	fmt.Fprintf(dbug, "meta published: '%s\n", metaPublished.String())
	fmt.Fprintf(dbug, "PUBLISHED: %d candidates\n", len(publishedCandidates))
	for _, c := range publishedCandidates {
		c.dump(dbug)
	}

	var published, updated fuzzytime.DateTime

	if len(publishedCandidates) > 0 {
		published = publishedCandidates[0].(*dateCandidate).dt
	} else if !metaPublished.Empty() {
		published = metaPublished
	} else if !urlDate.Empty() {
		published = urlDate
	}

	if metaUpdated.HasFullDate() {
		updated = metaUpdated
	} else if len(updatedCandidates) > 0 {
		updated = updatedCandidates[0].(*dateCandidate).dt
	}

	return published, updated
}
