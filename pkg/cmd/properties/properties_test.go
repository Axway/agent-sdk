package properties

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestAliasKeyPrefixResolvesNestedYamlValue guards against a regression where yaml nested under
// an agent's own name (e.g. "traceability_agent:") silently stopped resolving.
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
