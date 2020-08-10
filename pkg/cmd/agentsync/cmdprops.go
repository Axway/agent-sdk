package agentsync

import (
	"fmt"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/properties"
)

const syncFlag = "synchronize"

// CheckSyncFlag - checks to see if the sync flag was used and runs the ProcessSynchronization.
//   If return is 0 or greater exit should happen, with return as exitcode
func CheckSyncFlag(props properties.Properties) int {
	if props.BoolFlagValue(syncFlag) {
		// Call sync commands
		err := agentSync.ProcessSynchronization()
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
			return 1
		}
		return 0
	}
	return -1
}

// AddSyncConfigProperties - Adds the command properties needed for Sync Process Config
func AddSyncConfigProperties(props properties.Properties) {
	props.AddBoolFlag(syncFlag, "Run the sync process for the discovery agent")
}
