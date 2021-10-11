package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/filter"
	"github.com/Axway/agent-sdk/pkg/util"
	coreerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/exception"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	pathCategoryMapping      = "mappings"
	pathCategoryAutoCreation = "autocreation"
	conditionsKey            = "conditions"
	categoriesKey            = "categories"
	tagValueRegexStr         = "^(tag\\.)(.*)(\\.Value)$"
	categoryTitleRegExStr    = "[^a-zA-Z0-9_\\-\\(\\)\\[\\]\\s]+"
)

var autoCategoryCreation *bool
var isMappingConfigured *bool

// tagValueRegex - used to parse out the category name from the array of
var tagValueRegex = regexp.MustCompile(tagValueRegexStr)
var categoryTitleRegex = regexp.MustCompile(categoryTitleRegExStr)

// CategoryConfig - Interface to get category config
type CategoryConfig interface {
	IsAutocreationEnabled() bool
	IsConfigured() bool
	DetermineCategories(tags map[string]string) []string
	GetStaticCategories() []string
}

// CategoryConfiguration -
type CategoryConfiguration struct {
	Mappings         []*mapping `config:"mappings"`
	Autocreation     bool       `config:"autocreation"`
	staticCategories []string
	configured       bool // set to true if the mapping has been parsed
}

// mapping - the mapping defnition, if condition(s) match then categories are applied
type mapping struct {
	Condition  string        `config:"conditions" json:"conditions" yaml:"conditions"`
	Categories string        `config:"categories" json:"categories" yaml:"categories"`
	filter     filter.Filter // computed filter based on the conditions string
	categories []string      // all of the categories that are added for this filter, other than the variablized categories
	tagValues  []string      // list of all tags that we want the values to be categories
}

func (m *mapping) createFilter() error {
	var err error
	m.filter, err = filter.NewFilter(m.Condition)
	return err
}

func (m *mapping) setupCategories() error {
	var err error
	m.categories = make([]string, 0)
	m.tagValues = make([]string, 0)

	// Split the configured categories
	categories := strings.Split(m.Categories, ",")
	for _, cat := range categories {
		cat = strings.TrimSpace(cat)
		switch {
		case tagValueRegex.MatchString(cat):
			groups := tagValueRegex.FindStringSubmatch(cat)
			m.tagValues = append(m.tagValues, groups[2])
		default:
			m.categories = append(m.categories, cat)
		}
	}
	return err
}

// mappingStringToJSON - takes an input string, quoting any keys, then unmarshal it as a []mapping
func mappingStringToJSON(mappingString string) ([]*mapping, error) {
	var (
		categoryMapping []*mapping
		err             error
	)

	mappingString, err = util.RemoveUnquotedSpaces(mappingString)
	if err != nil {
		return categoryMapping, err
	}
	// try to unmarshal, if no error return now
	err = json.Unmarshal([]byte(mappingString), &categoryMapping)
	if err == nil {
		return categoryMapping, err
	}

	// find keys and quote them, if needed, so json.Unmarshal will work
	for _, key := range []string{conditionsKey, categoriesKey} {
		mappingString = strings.ReplaceAll(mappingString, fmt.Sprintf("%s:\"", key), fmt.Sprintf("\"%s\":\"", key))
	}

	err = json.Unmarshal([]byte(mappingString), &categoryMapping)
	return categoryMapping, err
}

// AddCategoryConfigProperties -
func AddCategoryConfigProperties(props properties.Properties, basePath string) {
	// mappings
	props.AddStringProperty(fmt.Sprintf("%s.%s", basePath, pathCategoryMapping), "", "Set mappings to use for the categories")

	// auto creation
	props.AddBoolProperty(fmt.Sprintf("%s.%s", basePath, pathCategoryAutoCreation), false, "Set to true to enable the creation of categories when they do not already exist")
}

// newCategoryConfig -
func newCategoryConfig() *CategoryConfiguration {
	cfg := &CategoryConfiguration{
		Mappings:         make([]*mapping, 0),
		Autocreation:     false,
		staticCategories: make([]string, 0),
		configured:       false,
	}
	// Set the global variables to point tot he configuration variables
	autoCategoryCreation = &cfg.Autocreation
	isMappingConfigured = &cfg.configured
	return cfg
}

// ParseCategoryConfig -
func ParseCategoryConfig(props properties.Properties, basePath string) CategoryConfig {
	cfg := newCategoryConfig()
	cfg.Autocreation = props.BoolPropertyValue(fmt.Sprintf("%s.%s", basePath, pathCategoryAutoCreation))

	// Determine the auth type
	categoryMapString := props.StringPropertyValue(fmt.Sprintf("%s.%s", basePath, pathCategoryMapping))
	if categoryMapString == "" {
		return cfg
	}
	categoryMappings, err := mappingStringToJSON(categoryMapString)
	if err != nil {
		exception.Throw(ErrBadConfig.FormatError(fmt.Sprintf("%s.%s", basePath, pathCategoryMapping)))
	}

	// Setup all filters based on conditions, separate the categories into an array
	for i, mapping := range categoryMappings {
		if err := mapping.createFilter(); err != nil {
			exception.Throw(coreerrors.Wrap(ErrBadConfig, err.Error()).FormatError(fmt.Sprintf("%s.%s[%d].%s", basePath, pathCategoryMapping, i, conditionsKey)))
		}
		mapping.setupCategories()
		cfg.staticCategories = append(cfg.staticCategories, mapping.categories...)
	}

	cfg.Mappings = categoryMappings
	cfg.staticCategories = util.RemoveDuplicateValuesFromStringSlice(cfg.staticCategories)
	cfg.configured = true
	return cfg
}

// IsCategoryAutocreationEnabled - return true when the global auto creation of categories is enabled
func IsCategoryAutocreationEnabled() bool {
	if autoCategoryCreation == nil {
		return false
	}
	return *autoCategoryCreation
}

// IsMappingConfigured - return true when category mappings have been configured
func IsMappingConfigured() bool {
	if isMappingConfigured == nil {
		return false
	}
	return *isMappingConfigured
}

// IsAutocreationEnabled - return true when the auto creation of categories is enabled
func (c *CategoryConfiguration) IsAutocreationEnabled() bool {
	return c.Autocreation
}

// IsConfigured - return true when the auto creation of categories is enabled
func (c *CategoryConfiguration) IsConfigured() bool {
	return c.configured
}

// GetStaticCategories - returns the array of the static categories for all the mappings
func (c *CategoryConfiguration) GetStaticCategories() []string {
	return c.staticCategories
}

// DetermineCategories - return a string array of all categories that match the configured conditions
func (c *CategoryConfiguration) DetermineCategories(tags map[string]string) []string {
	categories := make([]string, 0)
	if !c.configured {
		return categories
	}

	// Check all the mappings
	for _, mapping := range c.Mappings {
		// filter matched
		if mapping.filter.Evaluate(tags) {
			// append all categories that were just strings
			categories = append(categories, mapping.categories...)

			// get the values of all tags that were requested
			for _, tag := range mapping.tagValues {
				if value, ok := tags[tag]; ok {
					categories = append(categories, value)
				}
			}
		}
	}
	return util.RemoveDuplicateValuesFromStringSlice(c.processCategoryNames(categories))
}

// process all category names removing characters that are not allowed on Amplify Central
func (c *CategoryConfiguration) processCategoryNames(categories []string) []string {
	processedCategories := make([]string, 0)

	for _, categoryName := range categories {
		processedCategoryName := categoryTitleRegex.ReplaceAllString(categoryName, "")
		if processedCategoryName != categoryName {
			log.Warnf("Category names can only contain a-z, A-Z, 0-9, _, -, (), [], and space. Updating '%s' to '%s'", categoryName, processedCategoryName)
		}
		if processedCategoryName == "" {
			log.Warnf("Category name cannot be blank, skipping it")
			continue
		}
		processedCategories = append(processedCategories, processedCategoryName)
	}

	return processedCategories
}
