package arts

import (
	"regexp"
)

const namePat = `[\p{Lu}][\p{Ll}]+`
const surnamePrefixPat = `((?i)((van|van der|van de|van 't|von|de)\s)|d'|o'|mac|mc)`
const surnamePat = `((` + surnamePrefixPat + `)?(` + namePat + `)(-` + namePat + `){0,3})`
const fullNamePat = `^` + namePat + `(\s(` + surnamePat + `|([\p{Lu}][.]?))){0,3}\s` + surnamePat + `$`

var nameRE = regexp.MustCompile(fullNamePat)

//\s((([Vv]an|[Vv]an der|[Vv]an de|von)\s)([\p{Lu}][-\p{Ll}]+))?`)

func rateName(name string) float64 {
	name = compressSpace(name)
	if len(name) == 0 {
		return -1
	}

	if nameRE.MatchString(name) {
		return 1
	}
	/*
		parts := strings.Split(name, " ")
		if len(parts < 2) {
			return 0
		}
		if len(parts > 5) {
			return -1
		}
	*/
	return 0
}
