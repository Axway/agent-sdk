package util

import (
	"fmt"
	"regexp"
)

var (
	agentInfoRe   = regexp.MustCompile(`^([a-zA-Z]*)/(\d*.\d*.\d*)-([a-z0-9]*) SDK/(\d*.\d*.\d.*) ([a-z][a-zA-Z0-9-]*) ([a-z][a-zA-Z0-9-]*) (binary|docker) (.*grpc-go.*)$`)
	agentInfoReV2 = regexp.MustCompile(`^([a-zA-Z]*)\/(\d*.\d*.\d*)[-]?[a-z0-9]*? \(sdkVer:(\d*.\d*.\d.*)\; env:([a-z][a-zA-Z0-9-]*)\; agent:([a-z][a-zA-Z0-9-]*)\; reactive:(true|false)\) (.*grpc-go.*)$`)
)

type CentralUserAgent struct {
	AgentType           string `json:"type"`
	Version             string `json:"version"`
	CommitSHA           string `json:"sha"` // for backward compatibility
	SDKVersion          string `json:"sdkVersion,omitempty"`
	Environment         string `json:"environment,omitempty"`
	AgentName           string `json:"name,omitempty"`
	IsGRPC              bool   `json:"reactive"`
	UseGRPCStatusUpdate bool   `json:"-"`
}

func NewUserAgent(agentType, version, sdkVersion, environmentName, agentName string, isGRPC bool) *CentralUserAgent {
	return &CentralUserAgent{
		AgentType:   agentType,
		Version:     version,
		SDKVersion:  sdkVersion,
		Environment: environmentName,
		AgentName:   agentName,
		IsGRPC:      isGRPC,
	}
}

func (ca *CentralUserAgent) FormatUserAgent() string {
	ua := ""
	if ca.AgentType != "" && ca.Version != "" && ca.SDKVersion != "" {
		reactive := "false"
		if ca.IsGRPC {
			reactive = "true"
		}
		ua = fmt.Sprintf("%s/%s (sdkVer:%s; env:%s; agent:%s; reactive:%s)", ca.AgentType, ca.Version, ca.SDKVersion, ca.Environment, ca.AgentName, reactive)
	}
	return ua
}

func ParseUserAgent(userAgent string) *CentralUserAgent {
	matches := agentInfoReV2.FindStringSubmatch(userAgent)
	if len(matches) == 0 {
		// backward compatible user agent
		matches = agentInfoRe.FindStringSubmatch(userAgent)
		if len(matches) > 6 {
			return &CentralUserAgent{
				AgentType:   matches[1],
				Version:     matches[2],
				CommitSHA:   matches[3],
				SDKVersion:  matches[4],
				Environment: matches[5],
				AgentName:   matches[6],
			}
		}
	}
	if len(matches) > 5 {
		return &CentralUserAgent{
			AgentType:           matches[1],
			Version:             matches[2],
			SDKVersion:          matches[3],
			Environment:         matches[4],
			AgentName:           matches[5],
			UseGRPCStatusUpdate: true,
		}
	}

	return nil
}
