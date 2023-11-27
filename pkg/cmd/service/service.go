package service

import (
	"fmt"
	"os/exec"
	"strings"

	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/cmd/service/daemon"
)

var (
	// Name -
	Name string

	dependencies = []string{"network-online.target"}

	globalAgentService *AgentService

	execCommand = exec.Command
)

// AgentService -
type AgentService struct {
	service     daemon.Daemon
	Name        string
	Description string
	Path        string
	PathArg     string
	EnvFile     string
	User        string
	Group       string
	InstallDir  string
}

func newAgentService() (*AgentService, error) {
	service, err := daemon.New(Name, corecmd.BuildAgentDescription, dependencies...)
	if err != nil {
		return nil, err
	}

	return &AgentService{
		service:     service,
		Name:        Name,
		Description: corecmd.BuildAgentDescription,
	}, nil
}

// HandleServiceFlag - handles the action needed based on the service flag value
func (a *AgentService) HandleServiceFlag(command string) error {
	var err error
	var status string
	var serviceName string
	// complete the appropriate action for the service
	switch strings.ToLower(command) {
	case "install":
		fmt.Println("installing the agent service")
		fmt.Printf("service will look for config file at %s\n", a.Path)
		fmt.Printf("name of service to be installed: %s\n", a.service.GetServiceName())

		a.service.SetEnvFile(a.EnvFile)
		a.service.SetUser(a.User)
		a.service.SetGroup(a.Group)
		a.service.SetInstallDir(a.InstallDir)
		_, err = a.service.Install(a.PathArg, a.Path)
	case "update":
		fmt.Println("updating the agent service")

		a.service.SetEnvFile(a.EnvFile)
		a.service.SetUser(a.User)
		a.service.SetGroup(a.Group)
		a.service.SetInstallDir(a.InstallDir)
		_, err = a.service.Update(a.PathArg, a.Path)
	case "remove":
		fmt.Println("removing the agent service")
		_, err = a.service.Remove()
	case "start":
		fmt.Println("starting the agent service")
		_, err = a.service.Start()
	case "stop":
		fmt.Println("stopping the agent service")
		_, err = a.service.Stop()
	case "status":
		fmt.Println("getting the agent service status")
		status, err = a.service.Status()
	case "enable":
		fmt.Println("setting the agent to start on reboot")
		_, err = a.service.Enable()
	case "logs":
		var logs string
		fmt.Println("getting the service logs")
		logs, err = a.service.Logs()
		fmt.Println(logs)
	case "name":
		fmt.Println("getting the service name")
		serviceName = a.service.GetServiceName()
	default:
		err = fmt.Errorf("unknown value of '%s' given", command)
	}

	// error hit
	if err != nil {
		fmt.Printf("service %s command failed: %s\n", strings.ToLower(command), err.Error())
	} else {
		fmt.Printf("service %s command succeeded\n", strings.ToLower(command))
		if status != "" {
			fmt.Println(status)
		}
		if serviceName != "" {
			fmt.Println(serviceName)
		}
	}
	return err
}
