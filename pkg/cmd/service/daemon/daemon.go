package daemon

import "strings"

// Status constants.
const (
	statNotInstalled = "Service not installed"
)

// Daemon interface has a standard set of methods/commands
type Daemon interface {
	// GetTemplate - gets service config template
	GetTemplate() string

	// SetTemplate - sets service config template
	SetTemplate(string) error

	// Install the service into the system
	Install(args ...string) (string, error)

	// Update the service definition on the system
	Update(args ...string) (string, error)

	// Remove the service and all corresponding files from the system
	Remove() (string, error)

	// Start the service
	Start() (string, error)

	// Stop the service
	Stop() (string, error)

	// Status - check the service status
	Status() (string, error)

	// Enable - sets the service to persist reboots
	Enable() (string, error)

	// Logs - gets the service logs
	Logs() (string, error)

	// Run - run executable service
	Run(e Executable) (string, error)

	// SetEnvFile - sets the environment file argument for generating the agent command
	SetEnvFile(string) error

	// SetUser - sets the user that executes the service
	SetUser(string) error

	// setGroup - sets the group that executes the service
	SetGroup(string) error

	// GetServiceName - gets the name of the service
	GetServiceName() string

	// SetInstallDir - sets the installation directory
	SetInstallDir(string) error
}

// Executable interface defines controlling methods of executable service
type Executable interface {
	// Start - non-blocking start service
	Start()
	// Stop - non-blocking stop service
	Stop()
	// Run - blocking run service
	Run()
}

// New - Create a new daemon
//
// name: name of the service
//
// description: any explanation, what is the service, its purpose
func New(name, description string, dependencies ...string) (Daemon, error) {
	return newDaemon(strings.Join(strings.Fields(name), "_"), description, dependencies)
}
