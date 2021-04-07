package redaction

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

const (
	sanitizeValue = "{*}"
	http          = "http"
	https         = "https"
)

//Redactions - the public methods available for redaction config
type Redactions interface {
	URIRedaction(uri string) (string, error)
	PathRedaction(path string) string
	QueryArgsRedaction(queryArgs map[string][]string) (map[string][]string, error)
	QueryArgsRedactionString(queryArgs string) (string, error)
	RequestHeadersRedaction(requestHeaders map[string]string) (map[string]string, error)
	ResponseHeadersRedaction(responseHeaders map[string]string) (map[string]string, error)
}

// Config - the configuration of all redactions
type Config struct {
	Path            path   `config:"path" yaml:"path"`
	Args            filter `config:"queryArgument" yaml:"queryArgument"`
	RequestHeaders  filter `config:"requestHeader" yaml:"requestHeader"`
	ResponseHeaders filter `config:"responseHeader" yaml:"responseHeader"`
}

// path - the keyMatches to show, all else are redacted
type path struct {
	Allowed []show `config:"show" yaml:"show"`
}

// filter - the configuration of a filter for each redaction config
type filter struct {
	Allowed  []show     `config:"show" yaml:"show"`
	Sanitize []sanitize `config:"sanitize" yaml:"sanitize"`
}

// show - the keyMatches to show, all else are redacted
type show struct {
	KeyMatch string `config:"keyMatch" yaml:"keyMatch"`
}

// sanitize - the keys and values to sanitize
type sanitize struct {
	KeyMatch   string `config:"keyMatch" yaml:"keyMatch"`
	ValueMatch string `config:"valueMatch" yaml:"valueMatch"`
}

//redactionRegex - the compiled regex of the configuration fields
type redactionRegex struct {
	Redactions
	pathFilters           []showRegex
	argsFilters           filterRegex
	requestHeaderFilters  filterRegex
	responseHeaderFilters filterRegex
}

type filterRegex struct {
	show     []showRegex
	sanitize []sanitizeRegex
}

type showRegex struct {
	keyMatch *regexp.Regexp
}

type sanitizeRegex struct {
	keyMatch   *regexp.Regexp
	valueMatch *regexp.Regexp
}

//DefaultConfig - returns a default reaction config where all things are redacted
func DefaultConfig() Config {
	return Config{
		Path: path{
			Allowed: []show{},
		},
		Args: filter{
			Allowed:  []show{},
			Sanitize: []sanitize{},
		},
		RequestHeaders: filter{
			Allowed:  []show{},
			Sanitize: []sanitize{},
		},
		ResponseHeaders: filter{
			Allowed:  []show{},
			Sanitize: []sanitize{},
		},
	}
}

//SetupRedactions - set up redactionRegex based on the redactionConfig
func (cfg *Config) SetupRedactions() (Redactions, error) {
	var redactionSetup redactionRegex
	var err error

	// Setup the path filters
	redactionSetup.pathFilters, err = setupShowRegex(cfg.Path.Allowed)
	if err != nil {
		return nil, err
	}

	// Setup the arg filters
	redactionSetup.argsFilters.show, err = setupShowRegex(cfg.Args.Allowed)
	if err != nil {
		return nil, err
	}
	redactionSetup.argsFilters.sanitize, err = setupSanitizeRegex(cfg.Args.Sanitize)
	if err != nil {
		return nil, err
	}

	// Setup the request header filters
	redactionSetup.requestHeaderFilters.show, err = setupShowRegex(cfg.RequestHeaders.Allowed)
	if err != nil {
		return nil, err
	}
	redactionSetup.requestHeaderFilters.sanitize, err = setupSanitizeRegex(cfg.RequestHeaders.Sanitize)
	if err != nil {
		return nil, err
	}

	// Setup the response header filters
	redactionSetup.responseHeaderFilters.show, err = setupShowRegex(cfg.ResponseHeaders.Allowed)
	if err != nil {
		return nil, err
	}
	redactionSetup.responseHeaderFilters.sanitize, err = setupSanitizeRegex(cfg.ResponseHeaders.Sanitize)
	if err != nil {
		return nil, err
	}

	return &redactionSetup, err
}

// URIRedaction - takes a uri and returns the redacted version of that URI
func (r *redactionRegex) URIRedaction(fullURI string) (string, error) {
	parsedURL, err := url.ParseRequestURI(fullURI)
	if err != nil {
		return "", err
	}
	switch parsedURL.Scheme {
	case http, https, "":
		parsedURL.Path, err = PathRedaction(parsedURL.Path)
		if err != nil {
			return "", err
		}
		parsedURL.RawQuery, err = r.QueryArgsRedactionString(parsedURL.RawQuery)
		if err != nil {
			return "", err
		}
	}

	return url.QueryUnescape(parsedURL.String())
}

// PathRedaction - returns a string that has only allowed path elements
func (r *redactionRegex) PathRedaction(path string) string {
	pathSegments := strings.Split(path, "/")

	for i, segment := range pathSegments {
		if segment == "" {
			continue // skip blank segments
		}
		// If the value is not matched, sanitize it
		if !isValidValueToShow(segment, r.pathFilters) {
			pathSegments[i] = sanitizeValue
		}
	}

	return strings.Join(pathSegments, "/")
}

// QueryArgsRedaction - accepts a map[string][]string for arguments and returns the same map[string][]string with redacted
func (r *redactionRegex) QueryArgsRedaction(args map[string][]string) (map[string][]string, error) {
	queryArgs := url.Values{}

	for argName, argValue := range args {
		// First check for removals
		removed := false
		// If the name is not matched, remove it
		if !isValidValueToShow(argName, r.argsFilters.show) {
			removed = true
		}

		// Don't check for sanitization if arg was removed entirely
		if removed {
			continue
		}

		// Now check for sanitization
		runSanitize, sanitizeRegex := shouldSanitize(argName, r.argsFilters.sanitize)
		for _, value := range argValue {
			if runSanitize {
				queryArgs.Add(argName, sanitizeRegex.ReplaceAllLiteralString(value, sanitizeValue))
			} else {
				queryArgs.Add(argName, value)
			}
		}
	}

	return queryArgs, nil
}

// QueryArgsRedactionString - accepts a string for arguments and returns the same string with redacted
func (r *redactionRegex) QueryArgsRedactionString(args string) (string, error) {
	if args == "" {
		return "", nil // skip if there are no query args
	}

	var queryArgs map[string][]string

	err := json.Unmarshal([]byte(args), &queryArgs)
	if err != nil {
		return "", err
	}

	redactedArgs, err := r.QueryArgsRedaction(queryArgs)
	if err != nil {
		return "", err
	}

	queryArgsBytes, err := json.Marshal(redactedArgs)
	if err != nil {
		return "", err
	}

	return string(queryArgsBytes), nil
}

// RequestHeadersRedaction - accepts a string of response headers and returns the redacted and sanitize string
func (r *redactionRegex) RequestHeadersRedaction(headers map[string]string) (map[string]string, error) {
	return r.headersRedaction(headers, r.requestHeaderFilters)
}

// ResponseHeadersRedaction - accepts a string of response headers and returns the redacted and sanitize string
func (r *redactionRegex) ResponseHeadersRedaction(headers map[string]string) (map[string]string, error) {
	return r.headersRedaction(headers, r.responseHeaderFilters)
}

// headersRedaction - accepts a string of headers and the filters to apply then returns the redacted and sanitize string
func (r *redactionRegex) headersRedaction(headers map[string]string, filters filterRegex) (map[string]string, error) {
	newHeaders := make(map[string]string)

	for headerName, headerValue := range headers {
		// If the name is not matched, remove it
		if !isValidValueToShow(headerName, filters.show) {
			continue
		}

		newHeaders[headerName] = headerValue
		// Now check for sanitization
		if runSanitize, sanitizeRegex := shouldSanitize(headerName, filters.sanitize); runSanitize {
			newHeaders[headerName] = sanitizeRegex.ReplaceAllLiteralString(headerValue, sanitizeValue)
		}
	}

	return newHeaders, nil
}
