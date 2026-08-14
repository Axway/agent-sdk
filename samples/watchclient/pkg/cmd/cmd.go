package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/samples/watchclient/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var cfg = &client.Config{}

// NewRootCmd creates a new cobra.Command
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "server",
		Short:   "The Event Server for Control Plane",
		Version: "0.0.1",
		RunE:    run,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initViperConfig(cmd)
		},
	}

	initFlags(cmd)

	return cmd
}

func initFlags(cmd *cobra.Command) {
	cmd.Flags().String("tenant_id", "", "The org ID for watching the resource")
	cmd.Flags().String("host", "127.0.0.1", "The Service GRPC Host to connect")
	cmd.Flags().Uint32("port", 8000, "The Service GRPC port to connect")
	cmd.MarkFlagRequired("tenant_id")

	cmd.Flags().String("topic_self_link", "", "The self link of the WatchTopic")
	cmd.MarkFlagRequired("topic_self_link")

	cmd.Flags().String("auth.private_key", "./private_key.pem", "The private key associated with service account(default : ./private_key.pem)")
	cmd.Flags().String("auth.public_key", "./public_key.pem", "The public key associated with service account(default : ./public_key.pem)")
	cmd.Flags().String("auth.key_password", "", "The password for private key")
	cmd.Flags().String("auth.url", "https://login.axwaytest.net/auth", "The AxwayID auth URL")
	cmd.Flags().String("auth.client_id", "", "The service account client ID")
	cmd.Flags().Duration("auth.timeout", 10*time.Second, "The connection timeout for AxwayID")
	cmd.MarkFlagRequired("auth.client_id")

	cmd.Flags().Bool("insecure", false, "Do not verify the server cert on TLS connection")
	cmd.Flags().Bool("use_harvester", true, "Use harvester to sync events")
	cmd.Flags().String("harvester_host", "", "The Harvester Host")
	cmd.Flags().Uint32("harvester_port", 0, "The Harvester port")
	cmd.Flags().String("log_level", "info", "log level")
	cmd.Flags().String("log_format", "json", "line or json")
}

func initViperConfig(cmd *cobra.Command) error {
	v := viper.New()
	// All env vars must start with WC.
	v.SetEnvPrefix("wc")
	// Allows viper to tie a runtime flag, such as --auth.client_id to an env variable, AUTH_CLIENT_ID, and allows
	// viper to unmarshal values into nested structs. --auth.client_id -> config.auth.clientID
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	bindFlagsToViperConfig(cmd, v)

	err := v.Unmarshal(cfg)
	if err != nil {
		return err
	}

	return nil
}

// bindFlagsToViperConfig - For each flag, look up its corresponding env var, and use the env var if the flag is not set.
func bindFlagsToViperConfig(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		name := strings.ToUpper(f.Name)
		// Binds an environment variable to a viper arg. Ex: --tenant_id == TENANT_ID
		if err := v.BindPFlag(name, f); err != nil {
			panic(err)
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				panic(err)
			}
		}
	})
}

func run(_ *cobra.Command, _ []string) error {
	logger, err := getLogger(cfg.Level, cfg.Format)
	if err != nil {
		return err
	}

	wc, err := client.NewWatchClient(cfg, logger)
	if err != nil {
		return err
	}
	wc.Watch()
	return nil
}
