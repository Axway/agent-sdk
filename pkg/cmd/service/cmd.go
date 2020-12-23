package service

import (
	"fmt"
	"os"
	"strings"

	corecmd "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
	"github.com/spf13/cobra"
)

var argDescriptions = map[string]string{
	"install": "install the service, add --user and --group flags if necessary",
	"remove":  "remove the installed service",
	"start":   "start the installed service",
	"stop":    "stop the installed service",
	"status":  "get the status of the installed service",
	"logs":    "get the logs of the service",
	"enable":  "enable the service to persist on reboots of the OS",
	"name":    "get the name of the service",
}

// GenServiceCmd - generates the command version for a Beat.
func GenServiceCmd(pathArg string) *cobra.Command {
	// Create the validArgs array and the descriptions
	longDesc := ""
	validArgs := make([]string, 0, len(argDescriptions))
	for k := range argDescriptions {
		validArgs = append(validArgs, k)
		longDesc = fmt.Sprintf("%s\n%s\t\t%s", longDesc, k, argDescriptions[k])
	}
	shortDesc := fmt.Sprintf("Manage the OS service (%s)", strings.Join(validArgs, ", "))
	longDesc = fmt.Sprintf("%s\n%s", shortDesc, longDesc)

	cmd := &cobra.Command{
		Use:       "service command [flags]",
		Aliases:   []string{"svc"},
		ValidArgs: validArgs,
		Short:     shortDesc,
		Long:      longDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if globalAgentService == nil {
				var err error
				globalAgentService, err = newAgentService()
				if err != nil {
					return err
				}
			}

			if len(args) != 1 {
				log.Errorf("must provide only 1 arg to service (%s)", strings.Join(validArgs, ", "))
			}
			if _, ok := argDescriptions[args[0]]; !ok {
				log.Errorf("invalid command to service (%s)", strings.Join(validArgs, ", "))
			}
			globalAgentService.PathArg = fmt.Sprintf("--%s", pathArg)
			globalAgentService.Path = cmd.Flag(pathArg).Value.String()
			if pflag := cmd.Flag(corecmd.EnvFileFlag); pflag != nil {
				globalAgentService.EnvFile = pflag.Value.String()
			}
			if globalAgentService.Path == "." || globalAgentService.Path == "" {
				var err error
				globalAgentService.Path, err = os.Getwd()
				if err != nil {
					log.Errorf("error determining current working directory: %s", err.Error())
					return err
				}
			}
			globalAgentService.User = cmd.Flag("user").Value.String()
			globalAgentService.Group = cmd.Flag("group").Value.String()

			return globalAgentService.HandleServiceFlag(args[0])
		},
	}

	cmd.Flags().StringP("user", "u", "", "The OS user that will execute the service")
	cmd.Flags().StringP("group", "g", "", "The OS group that will execute the service")
	return cmd
}
