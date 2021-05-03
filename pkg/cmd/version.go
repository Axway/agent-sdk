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

// GetBuildDataPlaneType - returns the BuildDataPlaneType
func GetBuildDataPlaneType() string {
	if BuildDataPlaneType == "" {
		return "AgentSDK"
	}
	return BuildDataPlaneType
}
