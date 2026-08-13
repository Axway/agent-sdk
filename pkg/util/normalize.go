package util

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	separator = "-"
)

var (
	sharpSRegex       = regexp.MustCompile(`[ß]`)
	oSlashRegex       = regexp.MustCompile(`[Øø]`)
	aeRegex           = regexp.MustCompile(`[Ææ]`)
	oeRegex           = regexp.MustCompile(`[Œœ]`)
	invalidCharsRegex = regexp.MustCompile(`[^a-z0-9-]`)
	repeatedDashRegex = regexp.MustCompile(`[-]{2,}`)
)

func NormalizeNameForCentral(str string) string {
	if str == "" {
		return ""
	}

	str = strings.Trim(str, " ") // remove leading and trailing blanks

	// replace accented characters with their non-accented English character equivalents.
	str = removeDiacritics(str)
	str = sharpSRegex.ReplaceAllString(str, "ss")
	str = oSlashRegex.ReplaceAllString(str, "o")
	str = aeRegex.ReplaceAllString(str, "ae")
	str = oeRegex.ReplaceAllString(str, "oe")

	// make string all lowercase
	str = strings.ToLower(str)

	// replace invalid characters with "-" and reduce to 1 "-" maximum
	str = invalidCharsRegex.ReplaceAllString(str, separator)
	str = repeatedDashRegex.ReplaceAllString(str, separator)

	// remove leading and trailing "-"
	str = strings.Trim(str, separator)

	return str
}

type runesSet struct{}

func (s runesSet) Contains(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func removeDiacritics(input string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runesSet{}), norm.NFC)
	result, _, _ := transform.String(t, input)
	return result
}
