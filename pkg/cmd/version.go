package cmd

// BuildTime -
var BuildTime string

// BuildVersion -
var BuildVersion string

// BuildCommitSha -
var BuildCommitSha string

// BuildAgentName -
var BuildAgentName string

// BuildDataPlaneType -
var BuildDataPlaneType string

// agentSDKDataPlaneType - default data plane type when not set at build time
const agentSDKDataPlaneType = "AgentSDK"

// GetBuildDataPlaneType - returns the BuildDataPlaneType
func GetBuildDataPlaneType() string {
	if BuildDataPlaneType == "" {
		return agentSDKDataPlaneType
	}
	return BuildDataPlaneType
}
