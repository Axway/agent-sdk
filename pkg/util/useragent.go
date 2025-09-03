package util

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	agentInfoRe   = regexp.MustCompile(`^([a-zA-Z0-9]*)\/(\d*.\d*.\d*)[-]?([a-z0-9]*?) SDK/(\d*.\d*.\d.*) ([a-z][a-zA-Z0-9-]*) ([a-z][a-zA-Z0-9-.]*) (binary|docker)[ ]?(reactive)?`)
	agentInfoReV2 = regexp.MustCompile(`^([a-zA-Z0-9]*)\/(\d*.\d*.\d*)[-]?([-a-z0-9A-Z]*?) \(sdkVer:(\d*.\d*.\d.*)\; env:([a-zA-Z0-9-]*)\; agent:([a-zA-Z0-9-.]*)\; reactive:(true|false)\; hostname:([a-zA-Z0-9-_.]*)(; runtimeId:([a-zA-Z0-9-_.]*))?\) ??(grpc-go.*\/\d*.\d*.\d*)?$`)
)

type CentralUserAgent struct {
	AgentType           string `json:"type"`
	Version             string `json:"version"`
	CommitSHA           string `json:"sha,omitempty"` // for backward compatibility
	SDKVersion          string `json:"sdkVersion,omitempty"`
	Environment         string `json:"environment,omitempty"`
	AgentName           string `json:"name,omitempty"`
	IsGRPC              bool   `json:"reactive"`
	HostName            string `json:"hostname,omitempty"`
	UseGRPCStatusUpdate bool   `json:"-"`
	RuntimeId           string `json:"runtimeId,omitempty"`
}

func NewUserAgent(agentType, version, sdkVersion, environmentName, agentName string, isGRPC bool, runtimeId string) *CentralUserAgent {
	return &CentralUserAgent{
		AgentType:   agentType,
		Version:     strings.TrimPrefix(version, "v"),
		SDKVersion:  strings.TrimPrefix(sdkVersion, "v"),
		Environment: environmentName,
		AgentName:   agentName,
		IsGRPC:      isGRPC,
		RuntimeId:   runtimeId,
	}
}

func (ca *CentralUserAgent) FormatUserAgent() string {
	hostName, _ := os.Hostname()
	reactive := "false"
	if ca.IsGRPC {
		reactive = "true"
	}

	ua := fmt.Sprintf(
		"%s/%s (sdkVer:%s; env:%s; agent:%s; reactive:%s; hostname:%s; runtimeId:%s)",
		ca.AgentType,
		ca.Version,
		ca.SDKVersion,
		ca.Environment,
		ca.AgentName,
		reactive,
		hostName,
		ca.RuntimeId,
	)
	return ua
}

func ParseUserAgent(userAgent string) *CentralUserAgent {
	matches := agentInfoReV2.FindStringSubmatch(userAgent)
	if len(matches) == 0 {
		// backward compatible user agent
		matches = agentInfoRe.FindStringSubmatch(userAgent)
		if len(matches) > 6 {
			isGRPC := len(matches) > 8 && matches[8] == "reactive"
			return &CentralUserAgent{
				AgentType:   matches[1],
				Version:     matches[2],
				CommitSHA:   matches[3],
				SDKVersion:  matches[4],
				Environment: matches[5],
				AgentName:   matches[6],
				IsGRPC:      isGRPC,
			}
		}
	}
	if len(matches) > 9 {
		isGRPC := matches[7] == "true"
		return &CentralUserAgent{
			AgentType:           matches[1],
			Version:             matches[2],
			CommitSHA:           matches[3],
			SDKVersion:          matches[4],
			Environment:         matches[5],
			AgentName:           matches[6],
			IsGRPC:              isGRPC,
			HostName:            matches[8],
			UseGRPCStatusUpdate: isGRPC,
			RuntimeId:           matches[10],
		}
	}

	return nil
}
