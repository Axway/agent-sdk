package daemon

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Axway/agent-sdk/pkg/cmd"
)

const (
	systemctl     = "systemctl"
	journalctl    = "journalctl"
	serviceSuffix = ".service"
)

// systemDRecord - standard record (struct) for linux systemD version of daemon package
type systemDRecord struct {
	name         string
	description  string
	dependencies []string
	user         string
	group        string
	envFile      string
	installDir   string
}

// Standard service path for systemD daemons
func (s *systemDRecord) servicePath() string {
	return "/etc/systemd/system/" + s.serviceName()
}

// Is a service installed
func (s *systemDRecord) isInstalled() bool {

	if _, err := fs.Stat(s.servicePath()); err == nil {
		return true
	}

	return false
}

// Check service is running
func (s *systemDRecord) checkRunning() (string, bool) {
	output, err := execCmd(systemctl, "status", s.serviceName())
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

func (s *systemDRecord) serviceName() string {
	return s.name + serviceSuffix
}

// Install the service
func (s *systemDRecord) install(args ...string) (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return failed, err
	}

	srvPath := s.servicePath()

	if s.isInstalled() {
		return failed, ErrAlreadyInstalled.FormatError(s.serviceName())
	}

	file, err := fs.Create(srvPath)
	if err != nil {
		return failed, err
	}

	execPatch, err := executablePath(s.name)
	if err != nil {
		file.Close()
		return failed, err
	}

	templ, err := template.New("systemDConfig").Parse(systemDConfig)
	if err != nil {
		file.Close()
		return failed, err
	}

	if s.envFile != "" {
		args = append(args, fmt.Sprintf("--%s", cmd.EnvFileFlag), s.envFile)
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Dependencies, User, Group, Path, InstallDir, Args string
		}{
			s.name,
			s.description,
			strings.Join(s.dependencies, " "),
			s.user,
			s.group,
			s.installDir,
			execPatch,
			strings.Join(args, " "),
		},
	); err != nil {
		file.Close()
		return failed, err
	}

	if _, err := execCmd(systemctl, "daemon-reload"); err != nil {
		file.Close()
		return failed, err
	}

	file.Close()
	return success, nil
}

// Install the service
func (s *systemDRecord) Install(args ...string) (string, error) {
	installAction := "Install " + s.description + ":"

	msg, err := s.install(args...)
	return installAction + msg, err
}

// Update the service
func (s *systemDRecord) Update(args ...string) (string, error) {
	updateAction := "Updating " + s.description + ":"

	msg, err := s.remove()
	if err != nil {
		return updateAction + msg, err
	}

	msg, err = s.install(args...)
	return updateAction + msg, err
}

// Remove the service
func (s *systemDRecord) remove() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return failed, err
	}

	if !s.isInstalled() {
		return failed, ErrNotInstalled.FormatError(s.serviceName())
	}

	if _, ok := s.checkRunning(); ok {
		return failed, ErrCurrentlyRunning.FormatError(s.serviceName())
	}

	if _, err := execCmd(systemctl, "disable", s.serviceName()); err != nil {
		return failed, err
	}

	if err := fs.Remove(s.servicePath()); err != nil {
		return failed, err
	}

	return success, nil

}

// Remove the service
func (s *systemDRecord) Remove() (string, error) {
	removeAction := "Removing " + s.description + ":"

	msg, err := s.remove()
	return removeAction + msg, err
}

// Start the service
func (s *systemDRecord) Start() (string, error) {
	startAction := "Starting " + s.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return startAction + failed, err
	}

	if !s.isInstalled() {
		return startAction + failed, ErrNotInstalled.FormatError(s.serviceName())
	}

	if _, ok := s.checkRunning(); ok {
		return startAction + failed, ErrAlreadyRunning.FormatError(s.serviceName())
	}

	if _, err := execCmd(systemctl, "start", s.serviceName()); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (s *systemDRecord) Stop() (string, error) {
	stopAction := "Stopping " + s.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return stopAction + failed, err
	}

	if !s.isInstalled() {
		return stopAction + failed, ErrNotInstalled.FormatError(s.serviceName())
	}

	if _, ok := s.checkRunning(); !ok {
		return stopAction + failed, ErrAlreadyStopped.FormatError(s.serviceName())
	}

	if _, err := execCmd(systemctl, "stop", s.serviceName()); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (s *systemDRecord) Status() (string, error) {

	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if !s.isInstalled() {
		return statNotInstalled, ErrNotInstalled.FormatError(s.serviceName())
	}

	statusAction, _ := s.checkRunning()

	return statusAction, nil
}

// Logs - Get service logs
func (s *systemDRecord) Logs() (string, error) {

	if !s.isInstalled() {
		return statNotInstalled, ErrNotInstalled.FormatError(s.serviceName())
	}

	var data []byte
	var err error

	// run journalctl with --no-pager (get akk output), -b (logs on current boot only), -u service_name
	if data, err = execCmd(journalctl, "--no-pager", "-b", "-u", s.serviceName()); err != nil {
		return "", err
	}

	dataOutput := fmt.Sprintf("%s\nSee `journalctl -h` for alternative options to the `journalctl -u %s` command", string(data), s.serviceName())

	return dataOutput, nil
}

// Run - Run service
func (s *systemDRecord) Run(e Executable) (string, error) {
	runAction := "Running " + s.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

// Status - Get service status
func (s *systemDRecord) Enable() (string, error) {
	enableAction := "Enabling " + s.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return enableAction + failed, err
	}

	if !s.isInstalled() {
		return enableAction + failed, ErrNotInstalled.FormatError(s.serviceName())
	}

	if _, err := execCmd(systemctl, "enable", s.serviceName()); err != nil {
		return enableAction + failed, err
	}

	return enableAction + success, nil
}

// GetTemplate - gets service config template
func (s *systemDRecord) GetTemplate() string {
	return systemDConfig
}

// GetServiceName - gets service name
func (s *systemDRecord) GetServiceName() string {
	return s.serviceName()
}

// SetTemplate - sets service config template
func (s *systemDRecord) SetTemplate(tplStr string) error {
	systemDConfig = tplStr
	return nil
}

// SetEnvFile - sets the envFile that will be used by the service
func (s *systemDRecord) SetEnvFile(envFile string) error {
	// set the absolute path, incase it is relative
	envFileAbsolute, _ := filepath.Abs(envFile)
	s.envFile = envFileAbsolute
	return nil
}

// SetInstallDir - sets the installDir that will be used by the service
func (s *systemDRecord) SetInstallDir(installDir string) error {
	// set the absolute path, incase it is relative
	envFileAbsolute, _ := filepath.Abs(installDir)
	s.installDir = envFileAbsolute
	return nil
}

// SetUser - sets the user that will execute the service
func (s *systemDRecord) SetUser(user string) error {
	s.user = user
	return nil
}

// SetGroup - sets the group that will execute the service
func (s *systemDRecord) SetGroup(group string) error {
	s.group = group
	return nil
}

var systemDConfig = `[Unit]
Description={{.Description}}
Requires={{.Dependencies}}
After={{.Dependencies}}

[Service]
ExecStart={{.Path}} {{.Args}}
User={{.User}}
Group={{.Group}}
WorkingDirectory{{.InstallDir}}
Restart=on-failure
RestartSec=60s

[Install]
WantedBy=multi-user.target
`
