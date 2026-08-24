// Package specgen produces API specifications (MCP, A2A agent card, OAS3
// transport) from neutral, provider-agnostic inputs.
package specgen

import (
	_ "embed"
	"encoding/json"
	"strings"
)

const (
	// mcpVersion is the version of this spec format, not of the MCP protocol, so
	// it is fixed rather than caller-settable.
	mcpVersion = "1.0.0"

	defaultProtocolVersion = "2025-11-25"
	defaultVersion         = "1.0.0"
	defaultMode            = "text"

	gatewayURLPlaceholder = "__GATEWAY_URL__"
)

//go:embed oas3_http_stream.json
var oas3Template []byte

// Tool is a neutral tool definition, used both as input and, directly, as the
// tool element of an MCP spec. A nil InputSchema marshals as null; OutputSchema
// is omitted when absent.
type Tool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"inputSchema"`
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
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

// MCPSpecBuilder builds the spec document for an MCP server or gateway target.
// The protocol version and the capabilities are defaulted and can be overridden;
// the embedded OAS3 transport is derived from the endpoint URL.
type MCPSpecBuilder interface {
	SetName(name string) MCPSpecBuilder
	SetVersion(version string) MCPSpecBuilder
	SetEndpointURL(endpointURL string) MCPSpecBuilder
	// SetTools sets the server's tools. Leave unset when they cannot be
	// introspected, as for an MCP runtime target.
	SetTools(tools []Tool) MCPSpecBuilder
	SetProtocolVersion(protocolVersion string) MCPSpecBuilder
	SetCapabilities(capabilities MCPCapabilities) MCPSpecBuilder
	Build() *MCPSpec
}

type mcpSpecBuilder struct {
	name            string
	version         string
	endpointURL     string
	protocolVersion string
	capabilities    MCPCapabilities
	tools           []Tool
}

// NewMCPSpecBuilder returns a builder for an MCP spec.
func NewMCPSpecBuilder() MCPSpecBuilder {
	return &mcpSpecBuilder{
		version:         defaultVersion,
		protocolVersion: defaultProtocolVersion,
		capabilities:    MCPCapabilities{Tools: MCPToolsCapability{ListChanged: false}},
	}
}

func (b *mcpSpecBuilder) SetName(name string) MCPSpecBuilder {
	b.name = name
	return b
}

// SetVersion sets the server version. An empty version keeps the default.
func (b *mcpSpecBuilder) SetVersion(version string) MCPSpecBuilder {
	if version != "" {
		b.version = version
	}
	return b
}

func (b *mcpSpecBuilder) SetEndpointURL(endpointURL string) MCPSpecBuilder {
	b.endpointURL = endpointURL
	return b
}

func (b *mcpSpecBuilder) SetTools(tools []Tool) MCPSpecBuilder {
	b.tools = tools
	return b
}

// SetProtocolVersion overrides the MCP protocol version the server reports.
func (b *mcpSpecBuilder) SetProtocolVersion(protocolVersion string) MCPSpecBuilder {
	if protocolVersion != "" {
		b.protocolVersion = protocolVersion
	}
	return b
}

// SetCapabilities overrides the capabilities the server reports.
func (b *mcpSpecBuilder) SetCapabilities(capabilities MCPCapabilities) MCPSpecBuilder {
	b.capabilities = capabilities
	return b
}

// Build assembles the spec. With no tools set it yields the minimal transport
// with an empty tool list.
func (b *mcpSpecBuilder) Build() *MCPSpec {
	return &MCPSpec{
		MCP: mcpVersion,
		Details: MCPDetails{
			ProtocolVersion: b.protocolVersion,
			Capabilities:    b.capabilities,
			ServerInfo:      MCPServerInfo{Name: b.name, Version: b.version},
		},
		Tools: nonNilTools(b.tools),
		Transports: MCPTransports{
			HTTPStream: &HTTPStreamTransport{Definition: generateOAS3Spec(b.endpointURL)},
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

// AgentCardBuilder builds an A2A agent card. It produces the bare minimum card,
// for use when the real one cannot be fetched from the agent; everything beyond
// the defaults is set by the caller.
type AgentCardBuilder interface {
	SetName(name string) AgentCardBuilder
	SetDescription(description string) AgentCardBuilder
	SetVersion(version string) AgentCardBuilder
	SetEndpointURL(endpointURL string) AgentCardBuilder
	SetCapabilities(capabilities AgentCardCapabilities) AgentCardBuilder
	SetDefaultInputModes(modes []string) AgentCardBuilder
	SetDefaultOutputModes(modes []string) AgentCardBuilder
	// SetSkills sets the agent's skills. Leave unset when they are not known,
	// which is the case for a card that could not be fetched from the agent.
	SetSkills(skills []AgentCardSkill) AgentCardBuilder
	Build() *AgentCard
}

type agentCardBuilder struct {
	name               string
	description        string
	version            string
	endpointURL        string
	capabilities       AgentCardCapabilities
	defaultInputModes  []string
	defaultOutputModes []string
	skills             []AgentCardSkill
}

// NewAgentCardBuilder returns a builder for an A2A agent card.
func NewAgentCardBuilder() AgentCardBuilder {
	return &agentCardBuilder{
		version:            defaultVersion,
		defaultInputModes:  []string{defaultMode},
		defaultOutputModes: []string{defaultMode},
	}
}

func (b *agentCardBuilder) SetName(name string) AgentCardBuilder {
	b.name = name
	return b
}

func (b *agentCardBuilder) SetDescription(description string) AgentCardBuilder {
	b.description = description
	return b
}

// SetVersion sets the agent version. An empty version keeps the default.
func (b *agentCardBuilder) SetVersion(version string) AgentCardBuilder {
	if version != "" {
		b.version = version
	}
	return b
}

func (b *agentCardBuilder) SetEndpointURL(endpointURL string) AgentCardBuilder {
	b.endpointURL = endpointURL
	return b
}

func (b *agentCardBuilder) SetCapabilities(capabilities AgentCardCapabilities) AgentCardBuilder {
	b.capabilities = capabilities
	return b
}

func (b *agentCardBuilder) SetDefaultInputModes(modes []string) AgentCardBuilder {
	if len(modes) > 0 {
		b.defaultInputModes = modes
	}
	return b
}

func (b *agentCardBuilder) SetDefaultOutputModes(modes []string) AgentCardBuilder {
	if len(modes) > 0 {
		b.defaultOutputModes = modes
	}
	return b
}

func (b *agentCardBuilder) SetSkills(skills []AgentCardSkill) AgentCardBuilder {
	b.skills = skills
	return b
}

func (b *agentCardBuilder) Build() *AgentCard {
	return &AgentCard{
		Name:               b.name,
		Description:        b.description,
		URL:                b.endpointURL,
		Version:            b.version,
		Capabilities:       b.capabilities,
		DefaultInputModes:  b.defaultInputModes,
		DefaultOutputModes: b.defaultOutputModes,
		Skills:             nonNilSkills(b.skills),
	}
}

// generateOAS3Spec returns the static OAS3 JSON-RPC transport spec with
// servers[0].url set to endpointURL.
func generateOAS3Spec(endpointURL string) json.RawMessage {
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

// nonNilSkills returns skills, or an empty (non-nil) slice so the card's skills
// field marshals as [] rather than null.
func nonNilSkills(skills []AgentCardSkill) []AgentCardSkill {
	if skills == nil {
		return []AgentCardSkill{}
	}
	return skills
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
