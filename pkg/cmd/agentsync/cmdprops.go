package agentsync

import (
	"fmt"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/cmd/properties"
)

const syncFlag = "sync"

// CheckSyncFlag - checks to see if the sync flag was used, if so
func CheckSyncFlag(props properties.Properties) (bool, int) {
	if props.BoolFlagValue(syncFlag) {
		// Call sync commands
		err := agentSync.ProcessSynchronization()
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
			return true, 1
		}
		return true, 0
	}
	return false, 0
}

// AddSyncConfigProperties - Adds the command properties needed for Sync Process Config
func AddSyncConfigProperties(props properties.Properties) {
	props.AddBoolFlag(syncFlag, "Run the sync process for the discovery agent")
}
