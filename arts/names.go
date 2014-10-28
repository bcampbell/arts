package arts

import (
	"regexp"
	"strings"
)

/* euro-centric name patterns */
const namePat = `[\p{Lu}][\p{Ll}]+`
const surnamePrefixPat = `((?i)((van|van der|van de|van 't|von|de)\s)|d'|o'|mac|mc)`
const surnamePat = `((` + surnamePrefixPat + `)?(` + namePat + `)(-` + namePat + `){0,3})`
const fullNamePat = `^` + namePat + `(\s(` + surnamePat + `|([\p{Lu}][.]?))){0,3}\s` + surnamePat + `$`

var nameRE = regexp.MustCompile(fullNamePat)

// ad hoc list
var nameBlacklist = map[string]int{"facebook": 1, "tweet": 1, "widget": 1, "google": 1, "critic": 1, "reporter": 1, "follow": 1, "about": 1, "more": 1, "from": 1, "this": 1, "articles": 1, "crime": 1, "correspondent": 1}

//\s((([Vv]an|[Vv]an der|[Vv]an de|von)\s)([\p{Lu}][-\p{Ll}]+))?`)

func rateName(name string) float64 {
	name = compressSpace(name)
	if len(name) == 0 {
		return -1
	}

	var score float64
	if nameRE.MatchString(name) {
		score += 1
	}

	parts := strings.Fields(strings.ToLower(name))
	for _, part := range parts {
		if _, got := nameBlacklist[part]; got {
			score -= 1
		}
	}
	/*
		if len(parts) < 2 {
			return 0
		}
	*/
	if len(parts) > 5 {
		score -= 1
	}

	return score
}
