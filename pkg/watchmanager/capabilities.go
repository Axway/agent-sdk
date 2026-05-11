package watchmanager

import (
	"slices"

	proto "github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type Capability string

const (
	CapabilityPing Capability = "ping"
)

func SupportedCapabilities() []Capability {
	return []Capability{CapabilityPing}
}

func SetCapabilities(req *proto.Request, capabilities []Capability) {
	supported := SupportedCapabilities()
	supportedSet := make(map[Capability]struct{}, len(supported))
	for _, c := range supported {
		supportedSet[c] = struct{}{}
	}

	caps := make([]string, 0, len(capabilities))
	for _, c := range capabilities {
		if _, ok := supportedSet[c]; ok {
			caps = append(caps, string(c))
		}
	}

	if len(caps) == 0 {
		return
	}

	req.Capabilities = caps
}

func HasCapability(req *proto.Request, capability Capability) bool {
	if req.Capabilities == nil {
		return false
	}

	return slices.Contains(req.Capabilities, string(capability))
}
