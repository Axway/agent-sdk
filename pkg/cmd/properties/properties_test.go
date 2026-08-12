package properties

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestAliasKeyPrefixResolvesNestedYamlValue guards against the regression where an agent's
// yaml (nested entirely under its own name, e.g. "traceability_agent:") stopped resolving via
// the properties framework because the alias-key lookup was removed. Without SetAliasKeyPrefix,
// a property registered as "status.port" only ever reads the plain "status.port" key, which the
// nested yaml never populates, so it silently falls back to the flag's own default instead of
// the yaml's value.
func TestAliasKeyPrefixResolvesNestedYamlValue(t *testing.T) {
	yamlContent := []byte(`
traceability_agent:
  status:
    port: 8990
`)

	tests := map[string]struct {
		aliasKeyPrefix string
		expectedPort   int
	}{
		"no alias prefix set, falls back to flag default": {
			aliasKeyPrefix: "",
			expectedPort:   8989,
		},
		"alias prefix set, resolves nested yaml value": {
			aliasKeyPrefix: "traceability_agent",
			expectedPort:   8990,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			SetAliasKeyPrefix(tc.aliasKeyPrefix)
			defer SetAliasKeyPrefix("")

			rootCmd := &cobra.Command{Use: "test"}
			props := NewPropertiesWithSecretResolver(rootCmd, nil)
			props.AddIntProperty("status.port", 8989, "test port property")

			viper.SetConfigType("yaml")
			err := viper.ReadConfig(bytes.NewReader(yamlContent))
			assert.Nil(t, err)

			assert.Equal(t, tc.expectedPort, props.IntPropertyValue("status.port"))
		})
	}
}
