package daemon

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Service constants
const (
	success = "\t\t\t\t\t[  \033[32mOK\033[0m  ]" // Show colored "OK"
	failed  = "\t\t\t\t\t[\033[31mFAILED\033[0m]" // Show colored "FAILED"
)

// override for testing
var execCmd = execCommand

// execCommand - runs the command and returns the output
func execCommand(name string, arg ...string) ([]byte, error) {
	// #nosec
	cmd := exec.Command(name, arg...)

	// Set up byte buffers to read stdout
	var outbytes bytes.Buffer
	cmd.Stdout = &outbytes
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	return outbytes.Bytes(), nil
}

// ExecPath tries to get executable path
func ExecPath() (string, error) {
	return os.Executable()
}

// Lookup path for executable file
func executablePath(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		if _, err := fs.Stat(path); err == nil {
			return path, nil
		}
	}
	return os.Executable()
}

// Check root rights to use system service
func checkPrivileges() (bool, error) {

	if output, err := execCmd("id", "-g"); err == nil {
		if gid, parseErr := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32); parseErr == nil {
			if gid == 0 {
				return true, nil
			}
			return false, ErrRootPrivileges
		}
	}
	return false, ErrUnsupportedSystem
}
