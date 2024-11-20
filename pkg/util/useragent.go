package util

import (
	"fmt"
	"os"
	"regexp"
)

var (
	agentInfoRe   = regexp.MustCompile(`^([a-zA-Z]*)/(\d*.\d*.\d*)[-]?[a-z0-9]*? SDK/(\d*.\d*.\d.*) ([a-z][a-zA-Z0-9-]*) ([a-z][a-zA-Z0-9-]*) (binary|docker)[ ]?(reactive)?`)
	agentInfoReV2 = regexp.MustCompile(`^([a-zA-Z]*)\/(\d*.\d*.\d*)[-]?[a-z0-9]*? \(sdkVer:(\d*.\d*.\d.*)\; env:([a-zA-Z0-9-]*)\; agent:([a-zA-Z0-9-]*)\; reactive:(true|false)\; hostname:([a-zA-Z0-9-_]*)\)[ (.*grpc-go.*)$]?`)
)

type CentralUserAgent struct {
	AgentType           string `json:"type"`
	Version             string `json:"version"`
	CommitSHA           string `json:"sha"` // for backward compatibility
	SDKVersion          string `json:"sdkVersion,omitempty"`
	Environment         string `json:"environment,omitempty"`
	AgentName           string `json:"name,omitempty"`
	IsGRPC              bool   `json:"reactive"`
	HostName            string `json:"hostname,omitempty"`
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
	hostName, _ := os.Hostname()
	if ca.AgentType != "" && ca.Version != "" && ca.SDKVersion != "" {
		reactive := "false"
		if ca.IsGRPC {
			reactive = "true"
		}
		ua = fmt.Sprintf("%s/%s (sdkVer:%s; env:%s; agent:%s; reactive:%s; hostname:%s)", ca.AgentType, ca.Version, ca.SDKVersion, ca.Environment, ca.AgentName, reactive, hostName)
	}
	return ua
}

func ParseUserAgent(userAgent string) *CentralUserAgent {
	matches := agentInfoReV2.FindStringSubmatch(userAgent)
	if len(matches) == 0 {
		// backward compatible user agent
		matches = agentInfoRe.FindStringSubmatch(userAgent)
		if len(matches) > 6 {
			isGRPC := len(matches) > 7 && matches[7] == "reactive"
			return &CentralUserAgent{
				AgentType:   matches[1],
				Version:     matches[2],
				SDKVersion:  matches[3],
				Environment: matches[4],
				AgentName:   matches[5],
				IsGRPC:      isGRPC,
			}
		}
	}
	if len(matches) > 7 {
		isGRPC := matches[6] == "true"
		return &CentralUserAgent{
			AgentType:           matches[1],
			Version:             matches[2],
			SDKVersion:          matches[3],
			Environment:         matches[4],
			AgentName:           matches[5],
			IsGRPC:              isGRPC,
			HostName:            matches[7],
			UseGRPCStatusUpdate: isGRPC,
		}
	}

	return nil
}
