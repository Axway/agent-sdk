package daemon

import (
	"fmt"
)

// Get the daemon properly
func newDaemon(name, description string, dependencies []string) (Daemon, error) {
	// newer subsystem must be checked first
	if _, err := fs.Stat("/run/systemd/system"); err == nil {
		return &systemDRecord{name, description, dependencies, "", ""}, nil
	}
	return nil, fmt.Errorf("can not install service, need systemd")
}

// Get executable path
func execPath() (string, error) {
	return fs.Readlink("/proc/self/exe")
}
