package agentsync

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
)

const syncFlag = "synchronize"

var syncMode = false

// IsSyncMode - returns true when synchronize flag passed in
func IsSyncMode() bool {
	return syncMode
}

// SetSyncMode - checks for the syncFlag, if present sets IsSyncMode to true
func SetSyncMode(props properties.Properties) {
	if val := props.BoolFlagValue(syncFlag); val {
		syncMode = val
	}
}

// CheckSyncFlag - checks to see if the sync flag was used and runs the ProcessSynchronization.
//   If return is 0 or greater exit should happen, with return as exitcode
func CheckSyncFlag() int {
	if syncMode {
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

// AddSyncConfigProperties - Adds the flag needed for Sync Process Config
func AddSyncConfigProperties(props properties.Properties) {
	props.AddBoolFlag(syncFlag, "Run the sync process for the discovery agent")
}
