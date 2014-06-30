package arts

import (
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	//"github.com/matrixik/goquery"
	"code.google.com/p/cascadia"
	"regexp"
	//	"strings"
	//	"errors"
	//	"fmt"
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
	metaPublished   cascadia.Selector
	metaUpdated     cascadia.Selector
	tags            cascadia.Selector
	hatomPublished  cascadia.Selector
	hatomUpdated    cascadia.Selector
}{
	cascadia.MustCompile(`time, .published, .updated`),
	cascadia.MustCompile(`meta[property="article:published_time"], ` +
		`meta[name="dashboard_published_date"], ` +
		`meta[name="DC.date.issued"], ` +
		`meta[name="DCSext.articleFirstPublished"], ` +
		`meta[name="DCTERMS.created"]`),
	cascadia.MustCompile(`meta[property="article:modified_time"], ` +
		`meta[name="DCTERMS.modified"], ` +
		`meta[name="dashboard_updated_date"], ` +
		`meta[name="last-modified"]`),
	cascadia.MustCompile(`time,span,div`),
	//cascadia.MustCompile(`time,p,span,div,li,td,th,h4,h5,h6,font`),
	//cascadia.MustCompile(`span`),
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
	regexp.MustCompile(`(?i)published|posted`),
	// updatedIndicativeText
	regexp.MustCompile(`(?i)updated|last modified`),
	// urlDateFmts
	[]*regexp.Regexp{
		regexp.MustCompile(`/(?P<year>\d{4})/(?P<month>\d{2})/(?P<day>\d{2})/`),
		regexp.MustCompile(`/(?P<year>\d{4})/(?P<month>\d{2})/`),
		//		regexp.MustCompile(`[^0-9](?P<year>\d{4})-(?P<month>\d{1,2})-(?P<day>\d{1,2})[^0-9]`),
	},
	// genericClasses
	regexp.MustCompile(`(?i)updated|date|time|fecha`),
	// publishedClasses
	regexp.MustCompile(`(?i)published`),
	// updatedClasses
	regexp.MustCompile(`(?i)modified|updated`),
}

// dateFromURL looks for an obvious date in the url
func dateFromURL(artURL string) fuzzytime.Date {

	for _, pat := range datePats.urlDateFmts {
		m := pat.FindStringSubmatch(artURL)
		if len(m) < 3 {
			continue
		}
		var d fuzzytime.Date

		// year
		if foo, err := strconv.Atoi(m[1]); err == nil {
			d.SetYear(foo)
		} else {
			continue
		}

		//month
		if foo, err := strconv.Atoi(m[2]); err == nil {
			d.SetMonth(foo)
		} else {
			continue
		}

		// day (optional)
		if len(m) > 3 {
			if foo, err := strconv.Atoi(m[3]); err == nil {
				d.SetDay(foo)
			} else {
				continue
			}
		}
		// if we get this far, we've got enough
		return d
	}

	return fuzzytime.Date{}
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
	metaUpdated := fuzzytime.DateTime{}
	metaPublished := fuzzytime.DateTime{}

	for _, node := range dateSels.metaPublished.MatchAll(root) {
		content := getAttr(node, "content")
		metaPublished = fuzzytime.Extract(content)
		if metaPublished.HasFullDate() {
			break
		}
	}

	for _, node := range dateSels.metaUpdated.MatchAll(root) {
		content := getAttr(node, "content")
		metaUpdated = fuzzytime.Extract(content)
		if metaUpdated.HasFullDate() {
			break
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
func grabDates(root *html.Node, artURL string, contentNodes []*html.Node) (fuzzytime.DateTime, fuzzytime.DateTime) {
	dbug := Debug.DatesLogger
	var publishedCandidates = make(candidateList, 0, 32)
	var updatedCandidates = make(candidateList, 0, 32)

	// there might be an obvious date in the URL
	urlDate := dateFromURL(artURL)

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
			if urlDate.Conflicts(&dt.Date) {
				updatedC.addPoints(-1, "clash with date in url")
				publishedC.addPoints(-1, "clash with date in url")
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

	dbug.Printf("date from url: '%s\n", urlDate.String())
	dbug.Printf("meta updated: '%s\n", metaUpdated.String())
	dbug.Printf("meta published: '%s\n", metaPublished.String())

	publishedCandidates.Sort()
	dbug.Printf("PUBLISHED: %d candidates\n", len(publishedCandidates))
	for _, c := range publishedCandidates {
		c.dump(dbug)
	}

	updatedCandidates.Sort()
	dbug.Printf("UPDATED: %d candidates\n", len(updatedCandidates))
	for _, c := range updatedCandidates {
		c.dump(dbug)
	}

	var published, updated fuzzytime.DateTime

	// pick best candidate for published
	if best, err := publishedCandidates.Best(); err == nil {
		published = best.(*dateCandidate).dt
	} else {
		dbug.Printf("published: Didn't pick any (%s)", err)
	}

	if published.Empty() {
		if !metaPublished.Empty() {
			published = metaPublished
		} else if !urlDate.Empty() {
			published = fuzzytime.DateTime{Date: urlDate}
		}
	}

	// updated: use meta data if present
	if metaUpdated.HasFullDate() {
		updated = metaUpdated
	} else {
		if best, err := updatedCandidates.Best(); err == nil {
			updated = best.(*dateCandidate).dt
		} else {
			dbug.Printf("updated: Didn't pick any (%s)", err)
		}
	}

	return published, updated
}
