package byline

// tools for parsing bylines
// eg "By Matthew Garrahan in Los Angeles and Tim Bradshaw in San Francisco"
// name, location, job title, etc...

import (
	//	"golang.org/x/net/html"
	//	"fmt"
	"regexp"
	"strings"
	//	"unicode"
	//	"unicode/utf8"
)

// splitInclusive splits on a regexp, but the separators are included within the output strings
func splitInclusive(txt string, sep *regexp.Regexp) []string {
	matches := sep.FindAllStringSubmatchIndex(txt, -1)
	used := 0
	parts := make([]string, 0, len(matches)+1)
	for _, m := range matches {
		if used < m[0] {
			parts = append(parts, txt[used:m[0]])
		}
		used = m[0]
	}

	if used < len(txt) {
		parts = append(parts, txt[used:])
	}
	return parts
}

type kind int

const (
	kindUnknown kind = iota
	kindName
	kindJobTitle
	kindLocation
	kindPublication
	kindEmail
	kindSection
)

var jobTitleWords = map[string]struct{}{
	"editor":         struct{}{},
	"associate":      struct{}{},
	"reporter":       struct{}{},
	"correspondent":  struct{}{},
	"corespondent":   struct{}{},
	"director":       struct{}{},
	"writer":         struct{}{},
	"commentator":    struct{}{},
	"nutritionalist": struct{}{},
	"presenter":      struct{}{},
	"journalist":     struct{}{},
	"staff":          struct{}{},
	"cameraman":      struct{}{},
	"deputy":         struct{}{},
	"head":           struct{}{},
	"columnist":      struct{}{},
}

var locationWords = map[string]struct{}{
	"in":        struct{}{},
	"angeles":   struct{}{},
	"francisco": struct{}{},
	"london":    struct{}{},
}

var rejectWords = map[string]struct{}{
	"the": struct{}{},
}

// classify text as name, location, job title etc...
// TODO: cheesy hack for now. Try using a pretrained Naive Bayes thingy here instead!
func classify(txt string) kind {
	words := strings.Fields(txt)
	if len(words) == 0 {
		return kindUnknown
	}

	// cleanup
	for i := 0; i < len(words); i++ {
		words[i] = strings.TrimSpace(strings.ToLower(words[i]))
	}

	jtCnt := 0
	locCnt := 0
	rejectCnt := 0
	numCnt := 0

	for _, word := range words {
		if strings.ContainsAny(word, "0123456789") {
			numCnt++
		}
		if _, got := jobTitleWords[word]; got {
			jtCnt++
		}
		if _, got := locationWords[word]; got {
			locCnt++
		}
		if _, got := rejectWords[word]; got {
			rejectCnt++
		}
	}

	if rejectCnt > 0 {
		return kindUnknown
	}
	if numCnt > 0 {
		return kindUnknown // probably a date or time
	}

	if locCnt > 0 && locCnt >= jtCnt {
		return kindLocation
	}
	if jtCnt > 0 && jtCnt > locCnt {
		return kindJobTitle
	}

	return kindName
}

//bylineSplitPat = regexp.MustCompile(r`(?i)((?:\b(?:and|with|in)\b)|(?:[^-_.\w\s]+))`)
//   'indicative': re.compile(r'\s*\b(by|text by|posted by|written by|exclusive by|reviewed by|published by|photographs by|von)\b[:]?\s*',re.I)
/*
# n=name
# l=location
# t=jobtitle
# a=agency
# s=subject (eg "editor's briefing", "cricket")
# e=email address
*/

type Author struct {
	Name, Location, JobTitle, Email string
}

// regexp to split up parts of a byline
var bylineSplitPat = regexp.MustCompile(`(?i)\s*(?:,|(?:\b(?:by|text by|posted by|written by|exclusive by|reviewed by|published by|photographs by|and|by|in|for|special to|special for)\b))\s*`)
var fullStopPat = regexp.MustCompile(`[.]$`)

func Parse(txt string) []Author {
	out := []Author{}
	cur := Author{}
	parts := splitInclusive(txt, bylineSplitPat)
	// keep the splitting parts ("in", "and" etc) to aid the classifier
	// (eg "in" usually indicates a location)
	for _, part := range parts {

		cleaned := bylineSplitPat.ReplaceAllLiteralString(part, "")

		k := classify(part)
		//fmt.Printf("   '%s' => %v\n", part, k)
		switch k {
		case kindName:
			if cur.Name != "" {
				out = append(out, cur)
				cur = Author{}
			}
			// parts can end with fullstop, but we don't
			cleaned = fullStopPat.ReplaceAllLiteralString(cleaned, "")
			cur.Name = cleaned
		case kindJobTitle:
			cur.JobTitle = cleaned
		case kindLocation:
			cur.Location = cleaned
		case kindEmail:
			cur.Email = cleaned
		}
	}
	if cur.Name != "" {
		out = append(out, cur)
	}
	return out
}
