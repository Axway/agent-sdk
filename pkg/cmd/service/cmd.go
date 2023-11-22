package service

import (
	"fmt"
	"os"
	"strings"

	corecmd "github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/spf13/cobra"
)

var argDescriptions = map[string]string{
	"install": "install the service, add --user and --group flags if necessary",
	"update":  "update the service, add --user and --group flags if necessary",
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
				fmt.Printf("must provide exactly 1 arg to service command (%s)\n", strings.Join(validArgs, ", "))
				return nil
			}
			if _, ok := argDescriptions[args[0]]; !ok {
				fmt.Printf("invalid command to service (%s)\n", strings.Join(validArgs, ", "))
				return nil
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
					fmt.Printf("error determining current working directory: %s\n", err.Error())
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
