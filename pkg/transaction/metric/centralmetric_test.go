package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

func TestCentralMetricBuilderSetVersion(t *testing.T) {
	cases := map[string]struct {
		version string
	}{
		"sets version 3":    {version: "3"},
		"sets empty string": {version: ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewCentralMetricBuilder().SetVersion(tc.version)
			assert.Equal(t, tc.version, b.Build().Version)
		})
	}
}

func TestCentralMetricBuilderSetAPICDeployment(t *testing.T) {
	cases := map[string]struct {
		deployment string
	}{
		"sets prod":  {deployment: "prod"},
		"sets teams": {deployment: "teams"},
		"sets empty": {deployment: ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewCentralMetricBuilder().SetAPICDeployment(tc.deployment)
			assert.Equal(t, tc.deployment, b.Build().APICDeployment)
		})
	}
}

func TestCentralMetricBuilderSetEnvironmentRuntimeType(t *testing.T) {
	cases := map[string]struct {
		calls []string
		want  string
	}{
		"sets managed":              {calls: []string{"managed"}, want: "managed"},
		"sets unmanaged":            {calls: []string{"unmanaged"}, want: "unmanaged"},
		"sets unknown":              {calls: []string{"unknown"}, want: "unknown"},
		"overwrites on second call": {calls: []string{"managed", "unmanaged"}, want: "unmanaged"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewCentralMetricBuilder()
			for _, rt := range tc.calls {
				b.SetEnvironmentRuntimeType(rt)
			}
			result := b.Build()
			assert.NotNil(t, result.Environment)
			assert.Equal(t, tc.want, result.Environment.RuntimeType)
		})
	}
}

func TestCentralMetricBuilderSetAPIServiceRevision(t *testing.T) {
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
}

func TestCentralMetricBuilderSetReporter(t *testing.T) {
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
}
