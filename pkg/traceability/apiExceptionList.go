package traceability

import (
	"regexp"
	"strings"
)

func setUpAPIExceptionList(apiExceptionsList []string) (string, error) {
	// if set check for valid regex definitions
	if apiExceptionsList != nil {
		newShowRegex = make([]*regexp.Regexp, 0)

		// Get the api exceptions list
		exceptions := apiExceptionsList
		for i := range exceptions {
			exception := strings.TrimSpace(exceptions[i])

			// check for regex and then validate
			keyMatch, err := regexp.Compile(exception)
			if err != nil {
				return "", err
			}

			newShowRegex = append(newShowRegex, keyMatch)

		}
	}
	return "", nil
}

// getAPIExceptionsList - Returns traceability APIs exception list (api paths)
func getAPIExceptionsList() []*regexp.Regexp {
	if newShowRegex == nil {
		return []*regexp.Regexp{}
	}
	return newShowRegex
}

// newShowRegex - array of regexp.Regexp
var newShowRegex []*regexp.Regexp

// ShouldIgnoreEvent - check to see if the uri exists in exception list
func ShouldIgnoreEvent(uriRaw string) bool {
	exceptions := getAPIExceptionsList()

	// If the api path exists in the exceptions list, return true and ignore event
	for _, exception := range exceptions {
		if exception.MatchString(uriRaw) {
			return true
		}
	}

	// api path not found in exceptions list
	return false
}
