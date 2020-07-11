package service

import (
	"fmt"
	"os"
	"strings"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/common/cli"
)

var argDescriptions = map[string]string{
	"install": "install the service, add --user and --group flags if necessary",
	"remove":  "remove the installed service",
	"start":   "start the installed service",
	"stop":    "stop the installed service",
	"status":  "get the status of the installed service",
	"enable":  "enable the service to persist on reboots of the OS",
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
		Run: cli.RunWith(
			func(cmd *cobra.Command, args []string) error {
				if len(args) != 1 {
					log.Errorf("must provide only 1 arg to service (%s)", strings.Join(validArgs, ", "))
				}
				if _, ok := argDescriptions[args[0]]; !ok {
					log.Errorf("invalid command to service (%s)", strings.Join(validArgs, ", "))
				}
				GlobalAgentService.PathArg = fmt.Sprintf("--%s", pathArg)
				GlobalAgentService.Path = cmd.Flag(pathArg).Value.String()
				if GlobalAgentService.Path == "." || GlobalAgentService.Path == "" {
					var err error
					GlobalAgentService.Path, err = os.Getwd()
					if err != nil {
						log.Errorf("error determining current working directory: %s", err.Error())
					}
				}
				GlobalAgentService.User = cmd.Flag("user").Value.String()
				GlobalAgentService.Group = cmd.Flag("group").Value.String()

				GlobalAgentService.HandleServiceFlag(args[0])
				return nil
			}),
	}

	cmd.Flags().StringP("user", "u", "", "The OS user that will execute the service")
	cmd.Flags().StringP("group", "g", "", "The OS group that will execute the service")
	return cmd
}
