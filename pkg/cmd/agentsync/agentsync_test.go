package agentsync

import (
	"errors"
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type customAgentSync struct {
	AgentSync
	syncCalled bool
	err        string
}

func (c *customAgentSync) ProcessSynchronization() error {
	c.syncCalled = true
	if c.err != "" {
		return errors.New(c.err)
	}
	return nil
}

func setupCustomAgentSync(err string) *customAgentSync {
	// Create the custom agent sync object
	cas := &customAgentSync{
		syncCalled: false,
		err:        err,
	}

	// Set the agent sync implementation to be our custom implementation
	SetAgentSync(cas)

	return cas
}

func TestDefaultAgentSync(t *testing.T) {
	err := agentSync.ProcessSynchronization()

	assert.Nil(t, err, " There was an unexpected error thrown by the default ProcessSynchroniization method")
}

func TestCustomAgentSync(t *testing.T) {
	// Create the custom agent sync object
	cas := setupCustomAgentSync("")

	// Call the ProcessSynchronization
	err := agentSync.ProcessSynchronization()

	// Validate no err and that our method was called
	assert.Nil(t, err, "There was an unexpected error thrown by the default ProcessSynchroniization method")
	assert.True(t, cas.syncCalled, "The syncCalled attribute was not set properly, was our custom sync method used?")
}

func TestAddCmdProp(t *testing.T) {
	// Create command properties
	rootCmd := &cobra.Command{}
	props := properties.NewProperties(rootCmd)

	// Add the sync prop
	AddSyncConfigProperties(props)

	// Validate the property was added
	val := props.BoolPropertyValue(syncFlag)

	assert.False(t, val, "Validate that the default property value is false")
}

func SetSyncProperty(t *testing.T, sync bool) properties.Properties {
	// reset syncmode
	syncMode = false

	// Create command properties
	rootCmd := &cobra.Command{}
	props := properties.NewProperties(rootCmd)

	// Set the sync property to true
	props.AddBoolProperty(syncFlag, sync, "")
	SetSyncMode(props)

	// Validate the property was added
	val := props.BoolFlagValue(syncFlag)
	assert.Equal(t, sync, val, "Validate that the sync property was set appropriately")

	return props
}

func TestCheckSyncFlag(t *testing.T) {
	// Create the custom agent sync
	cas := setupCustomAgentSync("")

	// Set the property to true for the next test
	SetSyncProperty(t, true)

	// Call the CheckSyncFlag
	assert.False(t, cas.syncCalled, "The syncCalled attribute should be false prior to calling CheckSyncFlag")
	exitcode := CheckSyncFlag()
	assert.True(t, cas.syncCalled, "The syncCalled attribute was not set properly, was our custom sync method used?")
	assert.Equal(t, 0, exitcode, "Expected the sync process to have a 0 exitcode")

	// recreate the custom agent sync and props
	cas = setupCustomAgentSync("")

	// Set the property to false for the next test
	SetSyncProperty(t, false)

	// Call the CheckSyncFlag
	assert.False(t, cas.syncCalled, "The syncCalled attribute should be false prior to calling CheckSyncFlag")
	exitcode = CheckSyncFlag()
	assert.False(t, cas.syncCalled, "The syncCalled attribute false as expected")
	assert.Equal(t, -1, exitcode, "Expected the sync process to have a -1 exitcode")

	// recreate the custom agent sync and props
	cas = setupCustomAgentSync("failed")

	// Set the property to true for the next test
	SetSyncProperty(t, true)

	// Call the CheckSyncFlag
	assert.False(t, cas.syncCalled, "The syncCalled attribute should be false prior to calling CheckSyncFlag")
	exitcode = CheckSyncFlag()
	assert.True(t, cas.syncCalled, "The syncCalled attribute was not set properly, was our custom sync method used?")
	assert.Equal(t, 1, exitcode, "Expected the sync process to have a 1 exitcode")

}
