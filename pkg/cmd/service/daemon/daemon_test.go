package daemon

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const tempfile = "/tmp/tempfile"

type mockOSFS struct {
	statError    bool
	removeCalled bool
}

func (m *mockOSFS) Open(name string) (*os.File, error)   { return nil, nil }
func (m *mockOSFS) Readlink(name string) (string, error) { return "", nil }

func (m *mockOSFS) Create(name string) (*os.File, error) {
	return os.Create(tempfile)
}

func (m *mockOSFS) Remove(name string) error {
	m.removeCalled = true
	return nil
}

func (m *mockOSFS) Stat(name string) (os.FileInfo, error) {
	if m.statError {
		return nil, fmt.Errorf("error in stat")
	}
	return nil, nil
}

type fakeCommand struct {
	calls   int
	outputs []string
	cmds    []string
}

var fakeOutput fakeCommand

func fakeExecCommand(name string, arg ...string) ([]byte, error) {
	output := []byte(fakeOutput.outputs[fakeOutput.calls])
	fakeOutput.calls++
	fakeOutput.cmds = append(fakeOutput.cmds, fmt.Sprint(name, " ", strings.Join(arg, " ")))
	return output, nil
}

func TestNew(t *testing.T) {
	mOSFS := mockOSFS{}
	fs = &mOSFS

	daemon, err := New("daemon", "desc", "network")
	assert.Nil(t, err, "Error was not nil")
	assert.NotNil(t, daemon, "The daemon object was not returned")
	assert.IsType(t, &systemDRecord{}, daemon, "The returned type was incorrect")
}

func TestNewDaemon(t *testing.T) {
	mOSFS := mockOSFS{}
	fs = &mOSFS

	daemon, err := newDaemon("daemon", "desc", []string{"network"})
	assert.Nil(t, err, "Error was not nil")
	assert.NotNil(t, daemon, "The daemon object was not returned")
	assert.IsType(t, &systemDRecord{}, daemon, "The returned type was incorrect")

	mOSFS = mockOSFS{
		statError: true,
	}
	fs = &mOSFS

	daemon, err = newDaemon("daemon", "desc", []string{"network"})
	assert.NotNil(t, err, "An error was expected")
	assert.Nil(t, daemon, "The daemon interface was not expected")
}

func TestInstall(t *testing.T) {
	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"1000"},
	}
	mOSFS := mockOSFS{}
	fs = &mOSFS

	daemon, err := newDaemon("daemon", "desc", []string{"network"})
	assert.Nil(t, err, "Error was not nil")
	assert.NotNil(t, daemon, "The daemon object was not returned")

	output, err := daemon.Install()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.NotNil(t, err, "expected an error since we were not root")

	mOSFS = mockOSFS{
		statError: true,
	}
	fs = &mOSFS
	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"0", ""},
	}

	output, err = daemon.Install()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.Nil(t, err, "no error expected")
	assert.Len(t, fakeOutput.cmds, 2)
	assert.Equal(t, "systemctl daemon-reload", fakeOutput.cmds[1])
}

func TestRemove(t *testing.T) {
	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"1000"},
	}
	mOSFS := mockOSFS{}
	fs = &mOSFS

	daemon, err := newDaemon("daemon", "desc", []string{"network"})
	assert.Nil(t, err, "Error was not nil")
	assert.NotNil(t, daemon, "The daemon object was not returned")

	output, err := daemon.Remove()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.NotNil(t, err, "expected an error since we were not root")

	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"0", ""},
	}

	output, err = daemon.Remove()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.Nil(t, err, "no error expected")
	assert.Len(t, fakeOutput.cmds, 2)
	assert.Equal(t, "systemctl disable daemon.service", fakeOutput.cmds[1])
	assert.True(t, mOSFS.removeCalled, "Exppected a call to remove the service definition file")

}

func TestStart(t *testing.T) {
	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"1000"},
	}
	mOSFS := mockOSFS{}
	fs = &mOSFS

	daemon, err := newDaemon("daemon", "desc", []string{"network"})
	assert.Nil(t, err, "Error was not nil")
	assert.NotNil(t, daemon, "The daemon object was not returned")

	output, err := daemon.Start()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.NotNil(t, err, "expected an error since we were not root")

	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"0", "Active: active"},
	}

	output, err = daemon.Start()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.NotNil(t, err, "expected an error since the service would be 'running'")

	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"0", "Active: stopped", ""},
	}

	output, err = daemon.Start()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.Nil(t, err, "no error expected")
	assert.Len(t, fakeOutput.cmds, 3)
	assert.Equal(t, "systemctl start daemon.service", fakeOutput.cmds[2])
}

func TestStop(t *testing.T) {
	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"1000"},
	}
	mOSFS := mockOSFS{}
	fs = &mOSFS

	daemon, err := newDaemon("daemon", "desc", []string{"network"})
	assert.Nil(t, err, "Error was not nil")
	assert.NotNil(t, daemon, "The daemon object was not returned")

	output, err := daemon.Stop()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.NotNil(t, err, "expected an error since we were not root")

	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"0", "Active: stopped"},
	}

	output, err = daemon.Stop()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.NotNil(t, err, "expected an error since the service would be 'stopped'")

	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"0", "Active: active", ""},
	}

	output, err = daemon.Stop()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.Nil(t, err, "no error expected")
	assert.Len(t, fakeOutput.cmds, 3)
	assert.Equal(t, "systemctl stop daemon.service", fakeOutput.cmds[2])
}

func TestStatus(t *testing.T) {
	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"1000"},
	}
	mOSFS := mockOSFS{}
	fs = &mOSFS

	daemon, err := newDaemon("daemon", "desc", []string{"network"})
	assert.Nil(t, err, "Error was not nil")
	assert.NotNil(t, daemon, "The daemon object was not returned")

	output, err := daemon.Status()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.NotNil(t, err, "expected an error since we were not root")

	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"0", "Active: active"},
	}

	output, err = daemon.Status()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.Nil(t, err, "don't expect error for status command")
	assert.Len(t, fakeOutput.cmds, 2)
	assert.Equal(t, "systemctl status daemon.service", fakeOutput.cmds[1])
}

func TestEnabled(t *testing.T) {
	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"1000"},
	}
	mOSFS := mockOSFS{}
	fs = &mOSFS

	daemon, err := newDaemon("daemon", "desc", []string{"network"})
	assert.Nil(t, err, "Error was not nil")
	assert.NotNil(t, daemon, "The daemon object was not returned")

	output, err := daemon.Enable()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.NotNil(t, err, "expected an error since we were not root")

	execCmd = fakeExecCommand
	fakeOutput = fakeCommand{
		calls:   0,
		outputs: []string{"0", ""},
	}

	output, err = daemon.Enable()
	assert.NotNil(t, output, "Expected an output to be returned")
	assert.Nil(t, err, "no error expected")
	assert.Len(t, fakeOutput.cmds, 2)
	assert.Equal(t, "systemctl enable daemon.service", fakeOutput.cmds[1])
}
