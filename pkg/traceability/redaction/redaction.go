package redaction

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	defaultSanitizeValue = "{*}"
	http                 = "http"
	https                = "https"
)

var sanitizeValue string

// Redactions - the public methods available for redaction config
type Redactions interface {
	URIRedaction(uri string) (string, error)
	PathRedaction(path string) string
	QueryArgsRedaction(queryArgs map[string][]string) (map[string][]string, error)
	QueryArgsRedactionString(queryArgs string) (string, error)
	RequestHeadersRedaction(requestHeaders map[string]string) (map[string]string, error)
	ResponseHeadersRedaction(responseHeaders map[string]string) (map[string]string, error)
	JMSPropertiesRedaction(jmsProperties map[string]string) (map[string]string, error)
}

// Config - the configuration of all redactions
type Config struct {
	Path              Path   `config:"path" yaml:"path"`
	Args              Filter `config:"queryArgument" yaml:"queryArgument"`
	RequestHeaders    Filter `config:"requestHeader" yaml:"requestHeader"`
	ResponseHeaders   Filter `config:"responseHeader" yaml:"responseHeader"`
	MaskingCharacters string `config:"maskingCharacters" yaml:"maskingCharacters"`
	JMSProperties     Filter `config:"jmsProperties" yaml:"jmsProperties"`
}

// path - the keyMatches to show, all else are redacted
type Path struct {
	Allowed []Show `config:"show" yaml:"show"`
}

// filter - the configuration of a filter for each redaction config
type Filter struct {
	Allowed  []Show     `config:"show" yaml:"show"`
	Sanitize []Sanitize `config:"sanitize" yaml:"sanitize"`
}

// show - the keyMatches to show, all else are redacted
type Show struct {
	KeyMatch string `config:"keyMatch" yaml:"keyMatch"`
}

// sanitize - the keys and values to sanitize
type Sanitize struct {
	KeyMatch   string `config:"keyMatch" yaml:"keyMatch"`
	ValueMatch string `config:"valueMatch" yaml:"valueMatch"`
}

// redactionRegex - the compiled regex of the configuration fields
type redactionRegex struct {
	Redactions
	pathFilters           []showRegex
	argsFilters           filterRegex
	requestHeaderFilters  filterRegex
	responseHeaderFilters filterRegex
	jmsPropertiesFilters  filterRegex
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

// DefaultConfig - returns a default reaction config where all things are redacted
func DefaultConfig() Config {
	return Config{
		Path: Path{
			Allowed: []Show{},
		},
		Args: Filter{
			Allowed:  []Show{},
			Sanitize: []Sanitize{},
		},
		RequestHeaders: Filter{
			Allowed:  []Show{},
			Sanitize: []Sanitize{},
		},
		ResponseHeaders: Filter{
			Allowed:  []Show{},
			Sanitize: []Sanitize{},
		},
		MaskingCharacters: "{*}",
		JMSProperties: Filter{
			Allowed:  []Show{},
			Sanitize: []Sanitize{},
		},
	}
}

// SetupRedactions - set up redactionRegex based on the redactionConfig
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

	// Setup the jms properties filters
	redactionSetup.jmsPropertiesFilters.show, err = setupShowRegex(cfg.JMSProperties.Allowed)
	if err != nil {
		return nil, err
	}
	redactionSetup.jmsPropertiesFilters.sanitize, err = setupSanitizeRegex(cfg.JMSProperties.Sanitize)
	if err != nil {
		return nil, err
	}

	isValidMask, err := validateMaskingChars(cfg.MaskingCharacters)
	if err != nil {
		err = ErrInvalidRegex.FormatError("validate masking characters", cfg.MaskingCharacters, err)
		log.Error(err)
		return nil, err
	}

	if isValidMask {
		sanitizeValue = cfg.MaskingCharacters
	} else {
		log.Error("error validating masking characters: ", string(cfg.MaskingCharacters), ", using default mask: ", defaultSanitizeValue)
		sanitizeValue = defaultSanitizeValue
	}

	return &redactionSetup, err
}

// validateMaskingChars - validates the supplied masking character string against the accepted characters
func validateMaskingChars(mask string) (bool, error) {
	// available characters are alphanumeric, between 1-5 characters, and can contain '-' (hyphen), '*' (star), '#' (sharp), '^' (caret), '~' (tilde), '.' (dot), '{' (open curly bracket), '}' (closing curly bracket)
	regEx := "^([a-zA-Z0-9-*#^~.{}]){1,5}$"
	isMatch, err := regexp.MatchString(regEx, mask)

	return isMatch, err
}

// URIRedaction - takes a uri and returns the redacted version of that URI
func (r *redactionRegex) URIRedaction(fullURI string) (string, error) {
	// just in case uri is really a full url, we want to only want the URI portion
	parsedURI, err := url.ParseRequestURI(fullURI)
	if err != nil {
		return "", err
	}
	parsedURL, err := url.ParseRequestURI(parsedURI.RequestURI())
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

	queryArgs, _ := url.ParseQuery(args)

	redactedArgs, err := r.QueryArgsRedaction(queryArgs)
	if err != nil {
		return "", err
	}

	queryArgString := ""
	for key, val := range redactedArgs {
		if queryArgString != "" {
			queryArgString += "&"
		}
		queryArgString += fmt.Sprintf("%s=%s", key, strings.Join(val, ","))
	}

	return queryArgString, nil
}

// RequestHeadersRedaction - accepts a map of response headers and returns the redacted and sanitize map
func (r *redactionRegex) RequestHeadersRedaction(headers map[string]string) (map[string]string, error) {
	return r.headersRedaction(headers, r.requestHeaderFilters)
}

// ResponseHeadersRedaction - accepts a map of response headers and returns the redacted and sanitize map
func (r *redactionRegex) ResponseHeadersRedaction(headers map[string]string) (map[string]string, error) {
	return r.headersRedaction(headers, r.responseHeaderFilters)
}

// JMSPropertiesRedaction - accepts a map of JMS properties and returns the redacted and sanitize map
func (r *redactionRegex) JMSPropertiesRedaction(properties map[string]string) (map[string]string, error) {
	return r.headersRedaction(properties, r.jmsPropertiesFilters)
}

// headersRedaction - accepts a string of headers and the filters to apply then returns the redacted and sanitize map
func (r *redactionRegex) headersRedaction(properties map[string]string, filters filterRegex) (map[string]string, error) {
	newProperties := make(map[string]string)

	for propName, propValue := range properties {
		// If the name is not matched, remove it
		if !isValidValueToShow(propName, filters.show) {
			continue
		}

		newProperties[propName] = propValue
		// Now check for sanitization
		if runSanitize, sanitizeRegex := shouldSanitize(propName, filters.sanitize); runSanitize {
			newProperties[propName] = sanitizeRegex.ReplaceAllLiteralString(propValue, sanitizeValue)
		}
	}

	return newProperties, nil
}
