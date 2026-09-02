package specgen

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPSpecBuilder(t *testing.T) {
	inputSchemaA := json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}}}`)
	inputSchemaB := json.RawMessage(`{"type":"object","properties":{"y":{"type":"integer"}}}`)

	tests := map[string]struct {
		serverName  string
		endpointURL string
		tools       []Tool
		wantToolLen int
		assertSpec  func(t *testing.T, spec *MCPSpec)
	}{
		"top-level shape": {
			serverName:  "x-mcpvw2",
			endpointURL: "https://example.dev/",
			wantToolLen: 0,
			assertSpec: func(t *testing.T, spec *MCPSpec) {
				buf, err := json.Marshal(spec)
				require.NoError(t, err)
				var out map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(buf, &out))
				for _, key := range []string{"mcp", "details", "tools", "transports"} {
					_, ok := out[key]
					assert.Truef(t, ok, "expected top-level key %q", key)
				}
				assert.JSONEq(t, `"1.0.0"`, string(out["mcp"]))
			},
		},

		"details defaulted and populated from the builder": {
			serverName:  "my-gateway",
			endpointURL: "https://example.dev/",
			wantToolLen: 0,
			assertSpec: func(t *testing.T, spec *MCPSpec) {
				assert.Equal(t, defaultProtocolVersion, spec.Details.ProtocolVersion)
				assert.False(t, spec.Details.Capabilities.Tools.ListChanged)
				assert.Equal(t, "my-gateway", spec.Details.ServerInfo.Name)
				assert.Equal(t, "1.0.0", spec.Details.ServerInfo.Version)
			},
		},

		"tools array reflects the tools set": {
			serverName:  "gw",
			endpointURL: "https://example.dev/",
			tools: []Tool{
				{Name: "tool_a", Description: "desc A", InputSchema: inputSchemaA},
				{Name: "tool_b", Description: "desc B", InputSchema: inputSchemaB},
				{Name: "tool_c", Description: "desc C"},
			},
			wantToolLen: 3,
			assertSpec: func(t *testing.T, spec *MCPSpec) {
				assert.Equal(t, "tool_a", spec.Tools[0].Name)
				assert.Equal(t, "desc A", spec.Tools[0].Description)
				assert.Equal(t, inputSchemaA, spec.Tools[0].InputSchema)
				assert.Equal(t, "tool_b", spec.Tools[1].Name)
				assert.Equal(t, inputSchemaB, spec.Tools[1].InputSchema)
				assert.Equal(t, "tool_c", spec.Tools[2].Name)
				assert.Nil(t, spec.Tools[2].InputSchema, "missing InputSchema should serialize to null")
			},
		},

		"no tools (runtime MCP) renders tools as []": {
			serverName:  "gw",
			endpointURL: "https://example.dev/",
			assertSpec: func(t *testing.T, spec *MCPSpec) {
				assert.NotNil(t, spec.Tools, "tools must be non-nil so JSON renders as [] not null")
				buf, err := json.Marshal(spec)
				require.NoError(t, err)
				var out struct {
					Tools json.RawMessage `json:"tools"`
				}
				require.NoError(t, json.Unmarshal(buf, &out))
				assert.Equal(t, "[]", string(out.Tools))
			},
		},

		"transports carry embedded OAS3 with endpoint URL": {
			serverName:  "gw",
			endpointURL: "https://ssmogos-design.dev.10-129-144-202.nip.io:4443/",
			assertSpec: func(t *testing.T, spec *MCPSpec) {
				require.NotNil(t, spec.Transports.HTTPStream)
				var oas3 map[string]interface{}
				require.NoError(t, json.Unmarshal(spec.Transports.HTTPStream.Definition, &oas3))
				servers, ok := oas3["servers"].([]interface{})
				require.True(t, ok, "oas3 must have a servers array")
				require.Len(t, servers, 1)
				first, ok := servers[0].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "https://ssmogos-design.dev.10-129-144-202.nip.io:4443/", first["url"])
				assert.Equal(t, "3.0.1", oas3["openapi"])
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			spec := NewMCPSpecBuilder().
				SetName(tc.serverName).
				SetEndpointURL(tc.endpointURL).
				SetTools(tc.tools).
				Build()
			require.NotNil(t, spec)
			assert.Len(t, spec.Tools, tc.wantToolLen)
			if tc.assertSpec != nil {
				tc.assertSpec(t, spec)
			}
		})
	}
}

func TestMCPSpecBuilderOverrides(t *testing.T) {
	spec := NewMCPSpecBuilder().
		SetName("gw").
		SetVersion("2.3.4").
		SetEndpointURL("https://example.dev/").
		SetProtocolVersion("2026-01-01").
		SetCapabilities(MCPCapabilities{Tools: MCPToolsCapability{ListChanged: true}}).
		Build()

	assert.Equal(t, "2026-01-01", spec.Details.ProtocolVersion)
	assert.True(t, spec.Details.Capabilities.Tools.ListChanged)
	assert.Equal(t, "2.3.4", spec.Details.ServerInfo.Version)
	// The spec format version is not caller-settable.
	assert.Equal(t, mcpVersion, spec.MCP)
}

// An empty value must not wipe a default, so callers can pass through an
// optional field without having to check it first.
func TestMCPSpecBuilderEmptyOverridesKeepDefaults(t *testing.T) {
	spec := NewMCPSpecBuilder().
		SetVersion("").
		SetProtocolVersion("").
		Build()

	assert.Equal(t, defaultVersion, spec.Details.ServerInfo.Version)
	assert.Equal(t, defaultProtocolVersion, spec.Details.ProtocolVersion)
}

func TestAgentCardBuilder(t *testing.T) {
	card := NewAgentCardBuilder().
		SetName("my-agent").
		SetDescription("does things").
		SetEndpointURL("https://example.dev/mcp").
		Build()

	assert.Equal(t, "my-agent", card.Name)
	assert.Equal(t, "does things", card.Description)
	assert.Equal(t, "https://example.dev/mcp", card.URL)
	assert.Equal(t, "1.0.0", card.Version)
	assert.Equal(t, []string{"text"}, card.DefaultInputModes)
	assert.Equal(t, []string{"text"}, card.DefaultOutputModes)
	assert.False(t, card.Capabilities.Streaming)
	assert.NotNil(t, card.Skills, "skills must render as [] not null")
	assert.Empty(t, card.Skills)
}

func TestAgentCardBuilderOverrides(t *testing.T) {
	skills := []AgentCardSkill{{ID: "echo", Name: "Echo", Description: "repeats", Tags: []string{"demo"}}}

	card := NewAgentCardBuilder().
		SetName("agent").
		SetVersion("9.9.9").
		SetCapabilities(AgentCardCapabilities{Streaming: true}).
		SetDefaultInputModes([]string{"text/plain"}).
		SetDefaultOutputModes([]string{"application/json"}).
		SetSkills(skills).
		Build()

	assert.Equal(t, "9.9.9", card.Version)
	assert.True(t, card.Capabilities.Streaming)
	assert.Equal(t, []string{"text/plain"}, card.DefaultInputModes)
	assert.Equal(t, []string{"application/json"}, card.DefaultOutputModes)
	assert.Equal(t, skills, card.Skills)
}

func TestAgentCardBuilderEmptyOverridesKeepDefaults(t *testing.T) {
	card := NewAgentCardBuilder().
		SetVersion("").
		SetDefaultInputModes(nil).
		SetDefaultOutputModes([]string{}).
		Build()

	assert.Equal(t, defaultVersion, card.Version)
	assert.Equal(t, []string{"text"}, card.DefaultInputModes)
	assert.Equal(t, []string{"text"}, card.DefaultOutputModes)
}

func TestGenerateOAS3Spec(t *testing.T) {
	tests := map[string]struct {
		url     string
		wantURL string
	}{
		"plain gateway URL is placed verbatim in servers[0].url": {
			url:     "https://another.example/",
			wantURL: "https://another.example/",
		},
		"special characters in URL stay valid JSON after substitution": {
			// URLs shouldn't typically contain backslashes or quotes, but a
			// bad value must still produce valid JSON rather than corrupt
			// the embedded template.
			url:     `https://e"vil\example/`,
			wantURL: `https://e"vil\example/`,
		},
		"empty URL still produces valid OAS3": {
			url:     "",
			wantURL: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			raw := generateOAS3Spec(tc.url)

			var oas3 map[string]interface{}
			require.NoError(t, json.Unmarshal(raw, &oas3), "output must remain valid JSON")

			servers, _ := oas3["servers"].([]interface{})
			require.Len(t, servers, 1)
			first, _ := servers[0].(map[string]interface{})
			assert.Equal(t, tc.wantURL, first["url"])
		})
	}
}
