package daemon

import (
	"regexp"
	"strings"
	"text/template"
)

const (
	systemctl     = "systemctl"
	serviceSuffix = ".service"
)

// systemDRecord - standard record (struct) for linux systemD version of daemon package
type systemDRecord struct {
	name         string
	description  string
	dependencies []string
	user         string
	group        string
}

// Standard service path for systemD daemons
func (linux *systemDRecord) servicePath() string {
	return "/etc/systemd/system/" + linux.name + serviceSuffix
}

// Is a service installed
func (linux *systemDRecord) isInstalled() bool {

	if _, err := fs.Stat(linux.servicePath()); err == nil {
		return true
	}

	return false
}

// Check service is running
func (linux *systemDRecord) checkRunning() (string, bool) {
	output, err := execCmd(systemctl, "status", linux.name+serviceSuffix)
	if err == nil {
		if matched, err := regexp.MatchString("Active: active", string(output)); err == nil && matched {
			reg := regexp.MustCompile("Main PID: ([0-9]+)")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				return "Service (pid  " + data[1] + ") is running...", true
			}
			return "Service is running...", true
		}
	}

	return "Service is stopped", false
}

// Install the service
func (linux *systemDRecord) Install(args ...string) (string, error) {
	installAction := "Install " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return installAction + failed, err
	}

	srvPath := linux.servicePath()

	if linux.isInstalled() {
		return installAction + failed, ErrAlreadyInstalled
	}

	file, err := fs.Create(srvPath)
	if err != nil {
		return installAction + failed, err
	}

	execPatch, err := executablePath(linux.name)
	if err != nil {
		file.Close()
		return installAction + failed, err
	}

	templ, err := template.New("systemDConfig").Parse(systemDConfig)
	if err != nil {
		file.Close()
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Dependencies, User, Group, Path, Args string
		}{
			linux.name,
			linux.description,
			strings.Join(linux.dependencies, " "),
			linux.user,
			linux.group,
			execPatch,
			strings.Join(args, " "),
		},
	); err != nil {
		file.Close()
		return installAction + failed, err
	}

	if _, err := execCmd(systemctl, "daemon-reload"); err != nil {
		file.Close()
		return installAction + failed, err
	}

	file.Close()
	return installAction + success, nil
}

// Remove the service
func (linux *systemDRecord) Remove() (string, error) {
	removeAction := "Removing " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return removeAction + failed, err
	}

	if !linux.isInstalled() {
		return removeAction + failed, ErrNotInstalled
	}

	if _, err := execCmd(systemctl, "disable", linux.name+serviceSuffix); err != nil {
		return removeAction + failed, err
	}

	if err := fs.Remove(linux.servicePath()); err != nil {
		return removeAction + failed, err
	}

	return removeAction + success, nil
}

// Start the service
func (linux *systemDRecord) Start() (string, error) {
	startAction := "Starting " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return startAction + failed, err
	}

	if !linux.isInstalled() {
		return startAction + failed, ErrNotInstalled
	}

	if _, ok := linux.checkRunning(); ok {
		return startAction + failed, ErrAlreadyRunning
	}

	if _, err := execCmd(systemctl, "start", linux.name+serviceSuffix); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (linux *systemDRecord) Stop() (string, error) {
	stopAction := "Stopping " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return stopAction + failed, err
	}

	if !linux.isInstalled() {
		return stopAction + failed, ErrNotInstalled
	}

	if _, ok := linux.checkRunning(); !ok {
		return stopAction + failed, ErrAlreadyStopped
	}

	if _, err := execCmd(systemctl, "stop", linux.name+serviceSuffix); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (linux *systemDRecord) Status() (string, error) {

	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if !linux.isInstalled() {
		return statNotInstalled, ErrNotInstalled
	}

	statusAction, _ := linux.checkRunning()

	return statusAction, nil
}

// Run - Run service
func (linux *systemDRecord) Run(e Executable) (string, error) {
	runAction := "Running " + linux.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

// Status - Get service status
func (linux *systemDRecord) Enable() (string, error) {
	enableAction := "Enabling " + linux.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return enableAction + failed, err
	}

	if !linux.isInstalled() {
		return enableAction + failed, ErrNotInstalled
	}

	if _, err := execCmd(systemctl, "enable", linux.name+serviceSuffix); err != nil {
		return enableAction + failed, err
	}

	return enableAction + success, nil
}

// GetTemplate - gets service config template
func (linux *systemDRecord) GetTemplate() string {
	return systemDConfig
}

// SetTemplate - sets service config template
func (linux *systemDRecord) SetTemplate(tplStr string) error {
	systemDConfig = tplStr
	return nil
}

// SetUser - sets the user that will execute the service
func (linux *systemDRecord) SetUser(user string) error {
	linux.user = user
	return nil
}

// SetGroup - sets the group that will execute the service
func (linux *systemDRecord) SetGroup(group string) error {
	linux.group = group
	return nil
}

var systemDConfig = `[Unit]
Description={{.Description}}
Requires={{.Dependencies}}
After={{.Dependencies}}

[Service]
PIDFile=/var/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /var/run/{{.Name}}.pid
ExecStart={{.Path}} {{.Args}}
User={{.User}}
Group={{.Group}}
Restart=on-failure

[Install]
WantedBy=multi-user.target
`
