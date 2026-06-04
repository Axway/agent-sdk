package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

func TestCentralMetricBuilderNewSetters(t *testing.T) {
	t.Run("SetVersion", func(t *testing.T) {
		cases := map[string]struct {
			version string
		}{
			"sets version 3":   {version: "3"},
			"sets empty string": {version: ""},
		}
		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				b := NewCentralMetricBuilder().SetVersion(tc.version)
				assert.Equal(t, tc.version, b.Build().Version)
			})
		}
	})

	t.Run("SetAPICDeployment", func(t *testing.T) {
		cases := map[string]struct {
			deployment string
		}{
			"sets prod":   {deployment: "prod"},
			"sets teams":  {deployment: "teams"},
			"sets empty":  {deployment: ""},
		}
		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				b := NewCentralMetricBuilder().SetAPICDeployment(tc.deployment)
				assert.Equal(t, tc.deployment, b.Build().APICDeployment)
			})
		}
	})

	t.Run("SetEnvironmentRuntimeType", func(t *testing.T) {
		cases := map[string]struct {
			runtimeType string
		}{
			"sets managed":   {runtimeType: "managed"},
			"sets unmanaged": {runtimeType: "unmanaged"},
			"sets unknown":   {runtimeType: "unknown"},
		}
		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				b := NewCentralMetricBuilder().SetEnvironmentRuntimeType(tc.runtimeType)
				result := b.Build()
				assert.NotNil(t, result.Environment)
				assert.Equal(t, tc.runtimeType, result.Environment.RuntimeType)
			})
		}

		t.Run("creates Environment when nil", func(t *testing.T) {
			b := NewCentralMetricBuilder()
			assert.Nil(t, b.Build().Environment)
			b.SetEnvironmentRuntimeType("managed")
			assert.NotNil(t, b.Build().Environment)
		})

		t.Run("preserves existing Environment on second call", func(t *testing.T) {
			b := NewCentralMetricBuilder().
				SetEnvironmentRuntimeType("managed").
				SetEnvironmentRuntimeType("unmanaged")
			assert.Equal(t, "unmanaged", b.Build().Environment.RuntimeType)
		})
	})

	t.Run("SetAPIServiceRevision", func(t *testing.T) {
		ref := &models.ResourceReference{ID: "rev-123"}
		cases := map[string]struct {
			ref  *models.ResourceReference
			want *models.ResourceReference
		}{
			"sets reference": {ref: ref, want: ref},
			"sets nil":       {ref: nil, want: nil},
		}
		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				b := NewCentralMetricBuilder().SetAPIServiceRevision(tc.ref)
				assert.Equal(t, tc.want, b.Build().APIServiceRevision)
			})
		}
	})

	t.Run("SetReporter", func(t *testing.T) {
		reporter := &Reporter{AgentName: "test-agent", AgentVersion: "1.0.0"}
		cases := map[string]struct {
			reporter *Reporter
			want     *Reporter
		}{
			"sets reporter": {reporter: reporter, want: reporter},
			"sets nil":      {reporter: nil, want: nil},
		}
		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				b := NewCentralMetricBuilder().SetReporter(tc.reporter)
				assert.Equal(t, tc.want, b.Build().Reporter)
			})
		}
	})
}
