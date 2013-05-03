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

var pubdatePats = struct {
	urlDateFmts []*regexp.Regexp
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

var dateSels = struct {
	machineReadable cascadia.Selector
}{
	cascadia.MustCompile(`time, .published, .updated`),
}

func grabDates(root *html.Node, url string, dbug io.Writer) (string, error) {
	//	var candidates = make(CandidateMap)
	var candidates = make(CandidateList, 0, 32)

	urlDate := dateFromUrl(url)
	if !urlDate.Empty() {
		foo, err := urlDate.IsoFormat()
		if err == nil {
			return foo, nil
		}
	}

	// TODO: check for value-title pattern

	for _, node := range dateSels.machineReadable.MatchAll(root) {
		c := newCandidate(node, "")
		switch node.DataAtom {
		case atom.Time:
			s := getAttr(node, "datetime")
			if s == "" {
				s = getTextContent(node)
			}
			c.Txt = s
			c.addScore(2, "<time>")
			fmt.Printf("time: %s %s\n", describeNode(node), s)
		case atom.Abbr:
			s := getAttr(node, "title")
			if s == "" {
				s = getTextContent(node)
			}
			c.Txt = s
			fmt.Printf("abbr: %s %s\n", describeNode(node), s)
		}

		if c.TotalScore > 0 {
			candidates = append(candidates, c)
		}
	}

	fmt.Fprintf(dbug, "PUBDATE: %d candidates\n", len(candidates))
	for _, c := range candidates {
		c.dump(dbug)
	}

	return "", errors.New("No pubdate found")
}
