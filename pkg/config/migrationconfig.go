package config

import "github.com/Axway/agent-sdk/pkg/cmd/properties"

// MigrationConfig - Interface for migration settings config
type MigrationConfig interface {
	ShouldCleanInstances() bool
	validate()
}

// MigrationSettings -
type MigrationSettings struct {
	CleanInstances bool
}

func newMigrationConfig() MigrationConfig {
	return &MigrationSettings{
		CleanInstances: false,
	}
}

func newTestMigrationConfig() MigrationConfig {
	return &MigrationSettings{
		CleanInstances: true,
	}
}

func (m *MigrationSettings) validate() {
}

func (m *MigrationSettings) ShouldCleanInstances() bool {
	return m.CleanInstances
}

const (
	pathCleanInstances = "central.migration.cleanInstances"
)

// AddMigrationConfigProperties - Adds the command properties needed for Migration Config
func AddMigrationConfigProperties(props properties.Properties) {
	props.AddBoolProperty(pathCleanInstances, false, "Set this to clean all but latest instance, per stage, within an API Service")
}

// ParseMigrationConfig - Parses the Migration Config values from the command line
func ParseMigrationConfig(props properties.Properties) MigrationConfig {
	migrationConfig := newMigrationConfig()
	migrationSettings := migrationConfig.(*MigrationSettings)

	migrationSettings.CleanInstances = props.BoolPropertyValue(pathCleanInstances)

	return migrationSettings
}
