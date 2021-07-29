package cmd

// BuildTime -
var BuildTime string

// BuildVersion -
var BuildVersion string

// BuildCommitSha -
var BuildCommitSha string

// BuildAgentName - internal identification name for the agent
var BuildAgentName string

// BuildAgentDescription - agent name you wish to display in things like the version and help command
var BuildAgentDescription string

// BuildDataPlaneType -
var BuildDataPlaneType string

// SDKBuildVersion -
var SDKBuildVersion string

// agentSDKDataPlaneType - default data plane type when not set at build time
const agentSDKDataPlaneType = "AgentSDK"

// GetBuildDataPlaneType - returns the BuildDataPlaneType
func GetBuildDataPlaneType() string {
	if BuildDataPlaneType == "" {
		return agentSDKDataPlaneType
	}
	return BuildDataPlaneType
}
