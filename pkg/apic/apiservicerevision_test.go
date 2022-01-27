package apic

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestUpdateAPIServiceRevisionTitle(t *testing.T) {
	testCases := []struct {
		name     string
		format   string
		apiName  string
		stage    string
		label    string
		count    int
		expected string
	}{
		{
			name:     "No Stage",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}/\\d{2}/\\d{2} - r 1",
		},
		{
			name:     "Stage - default label",
			apiName:  "API-Name",
			stage:    "PROD",
			count:    5,
			expected: "API-Name \\(Stage\\: PROD\\) - \\d{4}/\\d{2}/\\d{2} - r 5",
		},
		{
			name:     "Stage - new label",
			apiName:  "API-Name",
			stage:    "PROD",
			label:    "Portal",
			count:    3,
			expected: "API-Name \\(Portal\\: PROD\\) - \\d{4}/\\d{2}/\\d{2} - r 3",
		},
		{
			name:     "Bad Date - default",
			format:   "{{.APIServiceName}} - {{.Date:YYY/MM/DD}} - r {{.Revision}}",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}/\\d{2}/\\d{2} - r 1",
		},
		{
			name:     "New Date Format",
			format:   "{{.APIServiceName}} - {{.Date:YYYY-MM-DD}}",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}-\\d{2}-\\d{2}",
		},
		{
			name:     "Deprecated Date",
			format:   "{{.APIServiceName}} - {{date}} - r {{.Revision}}",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}/\\d{2}/\\d{2} - r 1",
		},
		{
			name:     "Bar Variable - default",
			format:   "{{.APIServiceName1}} - {{date}} - r {{.Revision}}",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}/\\d{2}/\\d{2} - r 1",
		},
		{
			name:     "Stage - new format",
			format:   "{{.Stage}} - {{.APIServiceName}} - {{.Date:MM/DD/YYYY}} - r {{.Revision}}",
			apiName:  "API-Name",
			stage:    "MyStage",
			label:    "Test",
			count:    6,
			expected: "MyStage - API-Name - \\d{2}/\\d{2}/\\d{4} - r 6",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// create the service client
			c := ServiceClient{
				cfg: config.NewCentralConfig(config.DiscoveryAgent),
			}
			c.cfg.(*config.CentralConfiguration).APIServiceRevisionPattern = test.format

			s := &ServiceBody{
				APIName: test.apiName,
				Stage:   test.stage,
				serviceContext: serviceContext{
					revisionCount: test.count,
				},
				StageDescriptor: "Stage", // default
			}
			if test.label != "" {
				s.StageDescriptor = test.label
			}

			title := c.updateAPIServiceRevisionTitle(s)
			assert.Regexp(t, test.expected, title)
		})
	}
}
