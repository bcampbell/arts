package arts

import (
	"code.google.com/p/go.net/html"
	"code.google.com/p/go.net/html/atom"
	"fmt"
	//"github.com/matrixik/goquery"
	"code.google.com/p/cascadia"
	"regexp"
	//	"sort"
	"io"
	//	"strings"
	"errors"
	"github.com/bcampbell/fuzzytime"
	"strconv"
)

/*

   meta_dates = set()
   for meta in doc.findall('.//meta'):
       n = meta.get('name', meta.get('property', ''))
       if pats.pubdate['metatags'].search(n):
           logging.debug(" date: consider meta name='%s' content='%s'" % (n,meta.get('content','')))
           fuzzy = fuzzydate.parse_datetime(meta.get('content',''))
           if not fuzzy.empty_date():
               meta_dates.add(fuzzy.date(fuzzydate.fuzzydate(day=1)))
*/

/*
pubdate = {
    'metatags': re.compile('date|time',re.I),
    'classes': re.compile('published|updated|date|time|fecha',re.I),
    'url_datefmts': (
        re.compile(r'/(?P<year>\d{4})/(?P<month>\d{1,2})/(?P<day>\d{1,2})/',re.I),
        re.compile(r'[^0-9](?P<year>\d{4})-(?P<month>\d{1,2})-(?P<day>\d{1,2})[^0-9]',re.I),
        ),
    'comment_classes': re.compile('comment|respond',re.I),
    'pubdate_indicator': re.compile('published|posted|updated',re.I),
}
*/

type dateCandidate struct {
	standardCandidate
	dt fuzzytime.DateTime
}

func newDateCandidate(n *html.Node, txt string, dt fuzzytime.DateTime) candidate {
	return &dateCandidate{standardCandidate{n, txt, 0, 1, make([]string, 0, 4)}, dt}
}

var pubdatePats = struct {
	urlDateFmts []*regexp.Regexp // to get dates out of URLs
}{
	[]*regexp.Regexp{
		regexp.MustCompile(`/(?P<year>\d{4})/(?P<month>\d{1,2})/(?P<day>\d{1,2})/`),
		regexp.MustCompile(`[^0-9](?P<year>\d{4})-(?P<month>\d{1,2})-(?P<day>\d{1,2})[^0-9]`),
		// TODO: should accept YYYY/MM with missing day?
	},
}

// dateFromURl looks for an obvious date in the url
func dateFromUrl(url string) (d fuzzytime.Date) {

	for _, pat := range pubdatePats.urlDateFmts {
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

// machine readable times:
// express
// <time itemprop="datePublished" datetime="2013-05-05T21:35:22" class="published-date">

var dateSels = struct {
	machineReadable cascadia.Selector
	meta            cascadia.Selector
	tags            cascadia.Selector
	hatomPublished  cascadia.Selector
	hatomUpdated    cascadia.Selector
}{
	cascadia.MustCompile(`time, .published, .updated`),
	cascadia.MustCompile(`head meta`),
	cascadia.MustCompile(`time,p,span,div,li,td,th,h4,h5,h6,font`),
	cascadia.MustCompile(`hentry .published`),
	cascadia.MustCompile(`hentry .updated`),
}

var dateREs = struct {
	publishedIndicative *regexp.Regexp
	updatedIndicative   *regexp.Regexp
}{
	regexp.MustCompile(`published|posted`),
	regexp.MustCompile(`updated|last modified`),
}

//
func grabDates(root *html.Node, url string, dbug io.Writer) (string, error) {
	var publishedCandidates = make(candidateList, 0, 32)
	var updatedCandidates = make(candidateList, 0, 32)

	urlDate := dateFromUrl(url)
	if !urlDate.Empty() {
		foo, err := urlDate.IsoFormat()
		if err == nil {
			return foo, nil
		}
	}

	// check meta tags for anything useful
	// eg
	// <meta property="article:published_time" content="2013-05-02" />
	// <meta content="2013-05-05T11:30:09Z" property="article:modified_time">
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

	// TODO: other meta tags?
	// "DCSext.articleFirstPublished"
	// "DC.date.issued"
	// "last-modified"

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
		if dateREs.publishedIndicative.MatchString(txt) {
			publishedC.addPoints(1, "indicative text")
		}

		// TEST: hAtom date markup
		if dateSels.hatomPublished.Match(node) {
			publishedC.addPoints(2, "hentry .published")
		}
		if dateSels.hatomUpdated.Match(node) {
			publishedC.addPoints(2, "hentry .updated")
		}

		// TODO: check against meta tags and url
		/*
			if !metaPublished.Empty() && !dt.Conflicts(&metaPublished) {
				publishedC.addPoints(1, fmt.Sprintf("agrees with published date in <meta> %s vs %s", dt.String(), metaPublished.String()))
			}
			if !metaUpdated.Empty() && !dt.Conflicts(&metaUpdated) {
				updatedC.addPoints(1, "agrees with updated date in <meta>")
			}
		*/

		// TODO: indicative ids and classes

		// TODO: TEST - proximity to top or bottom of article content
		// TODO: check for value-title pattern?
		if publishedC.total() > 0 {
			publishedCandidates = append(publishedCandidates, publishedC)
		}

		if updatedC.total() > 0 {
			updatedCandidates = append(updatedCandidates, updatedC)
		}

	}

	fmt.Fprintf(dbug, "meta updated: '%s\n", metaUpdated.String())
	fmt.Fprintf(dbug, "UPDATED: %d candidates\n", len(updatedCandidates))
	for _, c := range updatedCandidates {
		c.dump(dbug)
	}

	fmt.Fprintf(dbug, "meta published: '%s\n", metaPublished.String())
	fmt.Fprintf(dbug, "PUBLISHED: %d candidates\n", len(publishedCandidates))
	for _, c := range publishedCandidates {
		c.dump(dbug)
	}

	return "", errors.New("No pubdate found")
}
