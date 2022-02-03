package traceability

import (
	"regexp"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// setUpAPIExceptionList - called from config to set up api exceptions list for traceability
func setUpAPIExceptionList(cfgAPIiExceptionsList []string) (string, error) {
	// if set check for valid regex definitions
	exceptionRegEx = make([]*regexp.Regexp, 0)

	// Get the api exceptions list
	exceptions := cfgAPIiExceptionsList
	for i := range exceptions {
		exception := strings.TrimSpace(exceptions[i])

		// check for regex and then validate
		keyMatch, err := regexp.Compile(exception)
		if err != nil {
			return exception, err
		}

		exceptionRegEx = append(exceptionRegEx, keyMatch)

	}

	return "", nil
}

// exceptionRegEx - array of regexp.Regexp
var exceptionRegEx []*regexp.Regexp

// ShouldIgnoreEvent - check to see if the uri exists in exception list
func ShouldIgnoreEvent(uriRaw string) bool {

	// If the api path exists in the exceptions list, return true and ignore event
	for _, exception := range exceptionRegEx {
		if exception.MatchString(uriRaw) {
			log.Debugf("%s found in exception list.  Do not process event.", uriRaw)
			return true
		}
	}

	// api path not found in exceptions list
	return false
}
