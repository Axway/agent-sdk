// Package specgen produces API specifications (MCP, A2A agent card, OAS3
// transport) from neutral, provider-agnostic inputs.
package specgen

import (
	_ "embed"
	"encoding/json"
	"strings"
)

const (
	mcpVersion         = "1.0.0"
	mcpProtocolVersion = "2025-11-25"
	defaultVersion     = "1.0.0"

	gatewayURLPlaceholder = "__GATEWAY_URL__"
)

//go:embed oas3_http_stream.json
var oas3Template []byte

// Input is the neutral description of a server/target a spec is generated for.
type Input struct {
	Name        string
	Description string
	EndpointURL string
	Version     string // optional; defaults to defaultVersion
	Tools       []Tool // empty for runtime MCP targets (tools can't be introspected)
}

// Tool is a neutral tool definition, used both as input and, directly, as the
// tool element of an MCP spec. A nil InputSchema marshals as null; OutputSchema
// is omitted when absent.
type Tool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"inputSchema"`
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
}

func (in Input) version() string {
	if in.Version != "" {
		return in.Version
	}
	return defaultVersion
}

// MCPSpec is the top-level spec document describing an MCP server.
type MCPSpec struct {
	MCP        string        `json:"mcp"`
	Details    MCPDetails    `json:"details"`
	Tools      []Tool        `json:"tools"`
	Transports MCPTransports `json:"transports"`
}

type MCPDetails struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    MCPCapabilities `json:"capabilities"`
	ServerInfo      MCPServerInfo   `json:"serverInfo"`
}

type MCPCapabilities struct {
	Tools MCPToolsCapability `json:"tools"`
}

type MCPToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type MCPServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MCPTransports struct {
	HTTPStream *HTTPStreamTransport `json:"http-stream,omitempty"`
}

type HTTPStreamTransport struct {
	Definition json.RawMessage `json:"definition"`
}

// GenerateMCPSpec builds an MCPSpec; empty in.Tools (e.g. an MCP runtime target)
// yields the minimal transport with an empty tool list.
func GenerateMCPSpec(in Input) *MCPSpec {
	return &MCPSpec{
		MCP: mcpVersion,
		Details: MCPDetails{
			ProtocolVersion: mcpProtocolVersion,
			Capabilities:    MCPCapabilities{Tools: MCPToolsCapability{ListChanged: false}},
			ServerInfo:      MCPServerInfo{Name: in.Name, Version: in.version()},
		},
		Tools: nonNilTools(in.Tools),
		Transports: MCPTransports{
			HTTPStream: &HTTPStreamTransport{Definition: GenerateOAS3Spec(in.EndpointURL)},
		},
	}
}

// AgentCard is a minimal A2A agent card (subset of the A2A spec) sufficient to
// register the runtime as an API service without backend access.
type AgentCard struct {
	Name               string                `json:"name"`
	Description        string                `json:"description"`
	URL                string                `json:"url"`
	Version            string                `json:"version"`
	Capabilities       AgentCardCapabilities `json:"capabilities"`
	DefaultInputModes  []string              `json:"defaultInputModes"`
	DefaultOutputModes []string              `json:"defaultOutputModes"`
	Skills             []AgentCardSkill      `json:"skills"`
}

type AgentCardCapabilities struct {
	Streaming bool `json:"streaming"`
}

type AgentCardSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// GenerateA2AAgentCard builds a bare-minimum agent card. Skills are left empty
// when the actual card cannot be fetched from the runtime.
func GenerateA2AAgentCard(in Input) *AgentCard {
	return &AgentCard{
		Name:               in.Name,
		Description:        in.Description,
		URL:                in.EndpointURL,
		Version:            in.version(),
		Capabilities:       AgentCardCapabilities{},
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Skills:             []AgentCardSkill{},
	}
}

// GenerateOAS3Spec returns the static OAS3 JSON-RPC transport spec with
// servers[0].url set to endpointURL.
func GenerateOAS3Spec(endpointURL string) json.RawMessage {
	out := strings.Replace(string(oas3Template), gatewayURLPlaceholder, jsonStringEscape(endpointURL), 1)
	return json.RawMessage(out)
}

// nonNilTools returns tools, or an empty (non-nil) slice so the spec's tools
// field marshals as [] rather than null.
func nonNilTools(tools []Tool) []Tool {
	if tools == nil {
		return []Tool{}
	}
	return tools
}

// jsonStringEscape escapes s for substitution into a JSON string literal
// (without the surrounding quotes).
func jsonStringEscape(s string) string {
	b, _ := json.Marshal(s)
	if len(b) < 2 {
		return s
	}
	return string(b[1 : len(b)-1])
}
