package redaction

import (
	"regexp"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Global Agent redactions
var agentRedactions Redactions

func init() {
	agentRedactions = &redactionRegex{}
}

//SetupGlobalRedaction - set up redactionRegex based on the redactionConfig
func SetupGlobalRedaction(cfg Config) error {
	var err error
	agentRedactions, err = cfg.SetupRedactions()
	return err
}

func setupShowRegex(showFilters []show) ([]showRegex, error) {
	newShowRegex := make([]showRegex, 0)
	for _, filter := range showFilters {
		if filter.KeyMatch == "" {
			continue // ignore blank keymatches as they match nothing
		}
		kc, err := regexp.Compile(filter.KeyMatch)
		if err != nil {
			log.Debug("Regex Error! Cannot Compile RequestHeader Remove Regex Keymatch", err)
			return []showRegex{}, err
		}

		newShowRegex = append(newShowRegex, showRegex{
			keyMatch: kc,
		})
	}
	return newShowRegex, nil
}

func setupSanitizeRegex(sanitizeFilters []sanitize) ([]sanitizeRegex, error) {
	newSanitizeRegex := make([]sanitizeRegex, 0)
	for _, filter := range sanitizeFilters {
		if filter.KeyMatch == "" {
			continue // ignore blank keymatches as they match nothing
		}
		kc, err := regexp.Compile(filter.KeyMatch)
		if err != nil {

			log.Debug("Regex Error! Cannot Compile RequestHeader Remove Regex Keymatch", err)
			return []sanitizeRegex{}, err
		}

		vc, err := regexp.Compile(filter.ValueMatch)
		if err != nil {
			log.Debug("Regex Error! Cannot Compile ArgsFilter Sanitize Regex Valuematch", err)
			return []sanitizeRegex{}, err

		}

		newSanitizeRegex = append(newSanitizeRegex, sanitizeRegex{
			keyMatch:   kc,
			valueMatch: vc,
		})
	}
	return newSanitizeRegex, nil
}

// URIRedaction - takes a uri and returns the redacted version of that URI
func URIRedaction(fullURI string) (string, error) {
	return agentRedactions.URIRedaction(fullURI)
}

// PathRedaction - returns a string that has only allowed path elements
func PathRedaction(path string) string {
	return agentRedactions.PathRedaction(path)
}

// QueryArgsRedaction - accepts a string for arguments and returns the same string with redacted
func QueryArgsRedaction(args map[string][]string) (map[string][]string, error) {
	return agentRedactions.QueryArgsRedaction(args)
}

// QueryArgsRedactionString - accepts a string for arguments and returns the same string with redacted
func QueryArgsRedactionString(args string) (string, error) {
	return agentRedactions.QueryArgsRedactionString(args)
}

// RequestHeadersRedaction - accepts a string of response headers and returns the redacted and sanitize string
func RequestHeadersRedaction(headers map[string]string) (map[string]string, error) {
	return agentRedactions.RequestHeadersRedaction(headers)
}

// ResponseHeadersRedaction - accepts a string of response headers and returns the redacted and sanitize string
func ResponseHeadersRedaction(headers map[string]string) (map[string]string, error) {
	return agentRedactions.ResponseHeadersRedaction(headers)
}

func isValidValueToShow(value string, matchers []showRegex) bool {
	for _, matcher := range matchers {
		if matcher.keyMatch.MatchString(value) {
			return true
		}
	}
	return false
}

func shouldSanitize(value string, matchers []sanitizeRegex) (bool, *regexp.Regexp) {
	for _, matcher := range matchers {
		if matcher.keyMatch.MatchString(value) {
			return true, matcher.valueMatch
		}
	}
	return false, nil
}
