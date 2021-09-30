package config

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const testBasePath = "test"

type expectedCategoryConfig struct {
	mappings         int
	autoCreation     bool
	staticCategories []string
	configured       bool
}

var defaultExpectedCategoryConfig = expectedCategoryConfig{
	mappings:         0,
	autoCreation:     false,
	staticCategories: make([]string, 0),
	configured:       false,
}

func validatecategoryconfig(t *testing.T, expVals expectedCategoryConfig, cfg *CategoryConfiguration) {
	assert.Len(t, cfg.Mappings, expVals.mappings)
	assert.Equal(t, expVals.autoCreation, cfg.IsAutocreationEnabled())
	assert.Equal(t, expVals.staticCategories, cfg.GetStaticCategories())
	assert.Equal(t, expVals.configured, cfg.IsConfigured())
	assert.Equal(t, expVals.autoCreation, IsCategoryAutocreationEnabled())
	assert.Equal(t, expVals.configured, IsMappingConfigured())
}

func TestCategoryConfig(t *testing.T) {
	// default config
	cfg := newCategoryConfig()
	assert.NotNil(t, cfg)
	validatecategoryconfig(t, defaultExpectedCategoryConfig, cfg)

	// update auto creation
	cfg = newCategoryConfig()
	rootCmd := &cobra.Command{}
	props := properties.NewProperties(rootCmd)
	props.AddBoolProperty(fmt.Sprintf("%s.%s", testBasePath, pathCategoryAutoCreation), true, "")
	cfg = ParseCategoryConfig(props, testBasePath).(*CategoryConfiguration)
	expected := defaultExpectedCategoryConfig
	expected.autoCreation = true
	validatecategoryconfig(t, expected, cfg)

	// set a mapping
	cfg = newCategoryConfig()
	rootCmd = &cobra.Command{}
	props = properties.NewProperties(rootCmd)
	props.AddBoolProperty(fmt.Sprintf("%s.%s", testBasePath, pathCategoryAutoCreation), true, "")
	props.AddStringProperty(fmt.Sprintf("%s.%s", testBasePath, pathCategoryMapping), `[{conditions:"tag.TagA.Exists()",categories:"CategoryA, CategoryB, tag.TagA.Value"}]`, "")
	cfg = ParseCategoryConfig(props, testBasePath).(*CategoryConfiguration)
	expected = defaultExpectedCategoryConfig
	expected.autoCreation = true
	expected.mappings = 1
	expected.staticCategories = []string{"CategoryA", "CategoryB"}
	expected.configured = true
	validatecategoryconfig(t, expected, cfg)
}

func TestDetermineCategories(t *testing.T) {
	// create mapping
	cfg := newCategoryConfig()
	rootCmd := &cobra.Command{}
	props := properties.NewProperties(rootCmd)
	props.AddBoolProperty(fmt.Sprintf("%s.%s", testBasePath, pathCategoryAutoCreation), true, "")
	props.AddStringProperty(fmt.Sprintf("%s.%s", testBasePath, pathCategoryMapping), `[{conditions:"tag.TagA.Exists()",categories:"CategoryA"},{conditions:"tag.TagB.Exists()",categories:"tag.TagB.Value"},{conditions:"tag.TagC == tagged",categories:"CategoryA"}]`, "")
	cfg = ParseCategoryConfig(props, testBasePath).(*CategoryConfiguration)

	testCases := []struct {
		name             string
		inputTags        map[string]string
		outputCategories []string
	}{
		{
			name: "No Categories",
			inputTags: map[string]string{
				"Tag": "tagValue",
			},
			outputCategories: []string{},
		},
		{
			name: "Static Category Match",
			inputTags: map[string]string{
				"TagA": "tagAValue",
			},
			outputCategories: []string{"CategoryA"},
		},
		{
			name: "Dynamic Category Match",
			inputTags: map[string]string{
				"TagB": "tagBValue1, tagBValue2",
			},
			outputCategories: []string{"tagBValue1 tagBValue2"},
		},
		{
			name: "Multiple Matches",
			inputTags: map[string]string{
				"TagA": "tagAValue",
				"TagB": "tagBValue",
			},
			outputCategories: []string{"CategoryA", "tagBValue"},
		},
		{
			name: "Tag Value No Match",
			inputTags: map[string]string{
				"TagC": "tagCValue",
			},
			outputCategories: []string{},
		},
		{
			name: "Tag Value Matches",
			inputTags: map[string]string{
				"TagC": "tagged",
			},
			outputCategories: []string{"CategoryA"},
		},
		{
			name: "One Category with Multiple Matches",
			inputTags: map[string]string{
				"TagA": "tagAValue",
				"TagC": "tagged",
			},
			outputCategories: []string{"CategoryA"},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			categories := cfg.DetermineCategories(test.inputTags)
			assert.Len(t, categories, len(test.outputCategories))
			for i, outputCategory := range test.outputCategories {
				assert.Equal(t, categories[i], outputCategory)
			}
		})
	}
}
