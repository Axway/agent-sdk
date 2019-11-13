package cmd

import (
	"fmt"
	"strings"
	"time"

	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	"github.com/spf13/cobra"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var agentName string

func CreateRootCmd(exeName, desc string, runFnnc func(cmd *cobra.Command, args []string) error) *cobra.Command {
	cobra.OnInitialize(initConfig)

	agentName = exeName

	var RootCmd = &cobra.Command{
		Use:     agentName,
		Short:   desc,
		Version: BuildVersion,
		RunE:    runFnnc,
	}

	// APIC Flags
	RootCmd.Flags().String("centralUrl", "https://platform.axway.com", "URL of API Central")
	RootCmd.Flags().String("centralTenantId", "", "Tenant ID for the owner of the environment")
	RootCmd.Flags().String("centralEnvironmentId", "", "Environment ID for the current environment")
	RootCmd.Flags().String("centralTeamId", "", "Team ID for the current default team for creating catalog")
	RootCmd.Flags().String("apiServerUrl", "", "The URL that the API Server is listening on")
	RootCmd.Flags().String("apiServerEnvironment", "", "The Environment that the APIs will be associated with in API Central")
	RootCmd.Flags().String("authPrivateKey", "/etc/private_key.pem", "Path to the private key for API Central Authentication")
	RootCmd.Flags().String("authPublicKey", "/etc/public_key", "Path to the public key for API Central Authentication")
	RootCmd.Flags().String("authKeyPassword", "", "Password for the private key, if needed")
	RootCmd.Flags().String("authUrl", "https://login-preprod.axway.com/auth", "API Central authentication URL")
	RootCmd.Flags().String("authRealm", "Broker", "API Central authentication Realm")
	RootCmd.Flags().String("authClientId", "", "Client ID for the service account")
	RootCmd.Flags().Duration("authTimeout", 10*time.Second, "Timeout waiting for AxwayID response")

	// APIC Lookups
	BindOrPanic("central.url", RootCmd.Flags().Lookup("centralUrl"))
	BindOrPanic("central.tenantId", RootCmd.Flags().Lookup("centralTenantId"))
	BindOrPanic("central.teamId", RootCmd.Flags().Lookup("centralEnvironmentId"))
	BindOrPanic("central.mode", RootCmd.Flags().Lookup("centralTeamId"))
	BindOrPanic("central.apiServerUrl", RootCmd.Flags().Lookup("apiServerUrl"))
	BindOrPanic("central.apiServerEnvironment", RootCmd.Flags().Lookup("apiServerEnvironment"))
	BindOrPanic("central.auth.privateKey", RootCmd.Flags().Lookup("authPrivateKey"))
	BindOrPanic("central.auth.publicKey", RootCmd.Flags().Lookup("authPublicKey"))
	BindOrPanic("central.auth.password", RootCmd.Flags().Lookup("authKeyPassword"))
	BindOrPanic("central.auth.url", RootCmd.Flags().Lookup("authUrl"))
	BindOrPanic("central.auth.realm", RootCmd.Flags().Lookup("authRealm"))
	BindOrPanic("central.auth.clientId", RootCmd.Flags().Lookup("authClientId"))
	BindOrPanic("central.auth.timeout", RootCmd.Flags().Lookup("authTimeout"))

	return RootCmd
}

func initConfig() {
	viper.SetConfigName(agentName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetTypeByDefaultValue(true)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func BindOrPanic(key string, flag *flag.Flag) {
	if err := viper.BindPFlag(key, flag); err != nil {
		panic(err)
	}
}

func ParseCentralConfig() *corecfg.CentralConfiguration {
	return &corecfg.CentralConfiguration{
		TenantID:         viper.GetString("central.tenantId"),
		TeamID:           viper.GetString("central.teamId"),
		Mode:             corecfg.StringAgentModeMap[strings.ToLower(viper.GetString("central.mode"))],
		APICDeployment:   viper.GetString("central.deployment"),
		EnvironmentName:  viper.GetString("central.environmenName"),
		EnvironmentID:    viper.GetString("central.environmentId"),
		URL:              viper.GetString("central.url"),
		APIServerVersion: viper.GetString("central.apiServerVersion"),
		Auth: &corecfg.AuthConfiguration{
			URL:        viper.GetString("central.auth.url"),
			Realm:      viper.GetString("central.auth.realm"),
			ClientID:   viper.GetString("central.auth.clientID"),
			PrivateKey: viper.GetString("central.auth.privateKey"),
			PublicKey:  viper.GetString("central.auth.publicKey"),
			KeyPwd:     viper.GetString("central.auth.keyPassword"),
			Timeout:    viper.GetDuration("central.auth.timeout") * time.Second,
		},
	}
}
