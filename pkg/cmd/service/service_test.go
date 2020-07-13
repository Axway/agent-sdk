package service

import (
	"bytes"
	"io/ioutil"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/takama/daemon"
)

func TestGenServiceCmd(t *testing.T) {
	cmd := GenServiceCmd("pathConfig")

	assert.NotNil(t, cmd, "The generated command was nil")
	assert.Contains(t, cmd.Short, "Manage the OS service")
	assert.Contains(t, cmd.Long, "Manage the OS service")
	assert.Contains(t, cmd.Long, argDescriptions["install"], "The install description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["remove"], "The remove description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["start"], "The start description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["stop"], "The stop description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["status"], "The status description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["enable"], "The enable description was not included in the long description")
}

func TestRunGenServiceCmd(t *testing.T) {
	Name = "discovery-agent"
	Description = "Discovery Agent Description"
	cmd := GenServiceCmd("pathConfig")

	// Capture the output from the cmd object
	b := bytes.NewBufferString("")
	cmd.SetOut(b)

	cmd.Flags().String("pathConfig", "", "")
	cmd.SilenceUsage = true

	cmd.SetArgs([]string{"install", "--pathConfig", "."})
	err := cmd.Execute()

	// Read the cmd object output
	out, _ := ioutil.ReadAll(b)

	assert.NotNil(t, err, "Error expected to be returned from command Execute")
	assert.Contains(t, string(out), "Error: You must have root user privileges. Possibly using 'sudo' command should help")
}

type mockDaemon struct {
	getTemplateCalled bool
	setTemplateCalled bool
	installCalled     bool
	removeCalled      bool
	startCalled       bool
	stopCalled        bool
	statusCalled      bool
	runCalled         bool
}

func (m *mockDaemon) GetTemplate() string {
	m.getTemplateCalled = true
	return ""
}

func (m *mockDaemon) SetTemplate(string) error {
	m.setTemplateCalled = true
	return nil
}

func (m *mockDaemon) Install(args ...string) (string, error) {
	m.installCalled = true
	return "", nil
}

func (m *mockDaemon) Remove() (string, error) {
	m.removeCalled = true
	return "", nil
}

func (m *mockDaemon) Start() (string, error) {
	m.startCalled = true
	return "", nil
}

func (m *mockDaemon) Stop() (string, error) {
	m.stopCalled = true
	return "", nil
}

func (m *mockDaemon) Status() (string, error) {
	m.statusCalled = true
	return "", nil
}

func (m *mockDaemon) Run(e daemon.Executable) (string, error) {
	m.runCalled = true
	return "", nil
}

func newMockAgentService() AgentService {
	return AgentService{
		service:     &mockDaemon{},
		Name:        "disco-agent",
		Description: "description",
		Path:        "/this/path",
		PathArg:     "--pathConfig",
		User:        "user",
		Group:       "group",
	}
}

func TestHandleService(t *testing.T) {
	var a AgentService
	var err error

	// Install
	a = newMockAgentService()
	err = a.HandleServiceFlag("install")
	assert.Nil(t, err, "Unexpected error returned")
	assert.True(t, a.service.(*mockDaemon).installCalled)
	assert.True(t, a.service.(*mockDaemon).setTemplateCalled)

	// Remove
	a = newMockAgentService()
	err = a.HandleServiceFlag("remove")
	assert.Nil(t, err, "Unexpected error returned")
	assert.True(t, a.service.(*mockDaemon).removeCalled)

	// Start
	a = newMockAgentService()
	err = a.HandleServiceFlag("start")
	assert.Nil(t, err, "Unexpected error returned")
	assert.True(t, a.service.(*mockDaemon).startCalled)

	// Stop
	a = newMockAgentService()
	err = a.HandleServiceFlag("stop")
	assert.Nil(t, err, "Unexpected error returned")
	assert.True(t, a.service.(*mockDaemon).stopCalled)

	// Status
	a = newMockAgentService()
	err = a.HandleServiceFlag("status")
	assert.Nil(t, err, "Unexpected error returned")
	assert.True(t, a.service.(*mockDaemon).statusCalled)

	// Enable
	a = newMockAgentService()
	var execCmd string
	var arguments []string

	// Catch the execCommand call
	execCommand = func(cmd string, args ...string) *exec.Cmd {
		execCmd = cmd
		arguments = args
		return &exec.Cmd{}
	}

	err = a.HandleServiceFlag("enable")
	assert.NotNil(t, err, "Expected error to be returned")
	assert.True(t, a.service.(*mockDaemon).statusCalled)
	assert.Equal(t, "systemctl", execCmd)
	assert.Len(t, arguments, 2)
}
