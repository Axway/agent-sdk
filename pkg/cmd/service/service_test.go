package service

import (
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/service/daemon"
	"github.com/stretchr/testify/assert"
)

const (
	UNEXPECTED_ERR = "Unexpected error returned"
)

type mockDaemon struct {
	installCalled     bool
	updateCalled      bool
	removeCalled      bool
	startCalled       bool
	stopCalled        bool
	statusCalled      bool
	logsCalled        bool
	runCalled         bool
	enableCalled      bool
	serviceNameCalled bool
}

func (m *mockDaemon) GetTemplate() string        { return "" }
func (m *mockDaemon) SetTemplate(string) error   { return nil }
func (m *mockDaemon) SetEnvFile(string) error    { return nil }
func (m *mockDaemon) SetUser(string) error       { return nil }
func (m *mockDaemon) SetGroup(string) error      { return nil }
func (m *mockDaemon) SetInstallDir(string) error { return nil }

func (m *mockDaemon) Install(args ...string) (string, error) {
	m.installCalled = true
	return "", nil
}

func (m *mockDaemon) Update(args ...string) (string, error) {
	m.updateCalled = true
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

func (m *mockDaemon) Logs() (string, error) {
	m.logsCalled = true
	return "", nil
}

func (m *mockDaemon) Run(e daemon.Executable) (string, error) {
	m.runCalled = true
	return "", nil
}

func (m *mockDaemon) Enable() (string, error) {
	m.enableCalled = true
	return "", nil
}

func (m *mockDaemon) GetServiceName() string {
	m.serviceNameCalled = true
	return ""
}

func newMockAgentService() *AgentService {
	return &AgentService{
		service:     &mockDaemon{},
		Name:        "disco-agent",
		Description: "description",
		Path:        "/this/path",
		PathArg:     "--pathConfig",
		User:        "user",
		Group:       "group",
		EnvFile:     "./filename",
		InstallDir:  "/this/install/dir",
	}
}

func TestGenServiceCmd(t *testing.T) {
	cmd := GenServiceCmd("pathConfig")

	assert.NotNil(t, cmd, "The generated command was nil")
	assert.Contains(t, cmd.Short, "Manage the OS service")
	assert.Contains(t, cmd.Long, "Manage the OS service")
	assert.Contains(t, cmd.Long, argDescriptions["install"], "The install description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["update"], "The update description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["remove"], "The remove description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["start"], "The start description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["stop"], "The stop description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["logs"], "The status description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["status"], "The status description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["enable"], "The enable description was not included in the long description")
	assert.Contains(t, cmd.Long, argDescriptions["name"], "The name description was not included in the long description")
}

func TestRunGenServiceCmd(t *testing.T) {
	Name = "discovery-agent"
	a := newMockAgentService()
	globalAgentService = a
	cmd := GenServiceCmd("pathConfig")

	cmd.Flags().String("pathConfig", "", "")
	cmd.SetArgs([]string{"install", "--pathConfig", "."})
	err := cmd.Execute()

	assert.Nil(t, err, "Error expected to be returned from command Execute")
	assert.True(t, a.service.(*mockDaemon).installCalled)

	os.Remove("./filename")
}

func TestHandleService(t *testing.T) {
	var a *AgentService
	var err error

	// Install
	a = newMockAgentService()
	err = a.HandleServiceFlag("install")
	assert.Nil(t, err, UNEXPECTED_ERR)
	assert.True(t, a.service.(*mockDaemon).installCalled)

	// Update
	a = newMockAgentService()
	err = a.HandleServiceFlag("update")
	assert.Nil(t, err, UNEXPECTED_ERR)
	assert.True(t, a.service.(*mockDaemon).updateCalled)

	// Remove
	a = newMockAgentService()
	err = a.HandleServiceFlag("remove")
	assert.Nil(t, err, UNEXPECTED_ERR)
	assert.True(t, a.service.(*mockDaemon).removeCalled)

	// Start
	a = newMockAgentService()
	err = a.HandleServiceFlag("start")
	assert.Nil(t, err, UNEXPECTED_ERR)
	assert.True(t, a.service.(*mockDaemon).startCalled)

	// Stop
	a = newMockAgentService()
	err = a.HandleServiceFlag("stop")
	assert.Nil(t, err, UNEXPECTED_ERR)
	assert.True(t, a.service.(*mockDaemon).stopCalled)

	// Logs
	a = newMockAgentService()
	err = a.HandleServiceFlag("logs")
	assert.Nil(t, err, UNEXPECTED_ERR)
	assert.True(t, a.service.(*mockDaemon).logsCalled)

	// Status
	a = newMockAgentService()
	err = a.HandleServiceFlag("status")
	assert.Nil(t, err, UNEXPECTED_ERR)
	assert.True(t, a.service.(*mockDaemon).statusCalled)

	// Enable
	a = newMockAgentService()
	err = a.HandleServiceFlag("enable")
	assert.Nil(t, err, UNEXPECTED_ERR)
	assert.True(t, a.service.(*mockDaemon).enableCalled)

	// Service Name
	a = newMockAgentService()
	err = a.HandleServiceFlag("name")
	assert.Nil(t, err, UNEXPECTED_ERR)
	assert.True(t, a.service.(*mockDaemon).serviceNameCalled)
}
