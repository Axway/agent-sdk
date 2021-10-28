package cmd

import (
	"strings"
	"time"

	"github.com/Axway/agent-sdk/samples/watchclient/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var logger *logrus.Logger
var log logrus.FieldLogger = logrus.StandardLogger()

// RootCmd configures the command params of the csa
var RootCmd = &cobra.Command{
	Use:     "server",
	Short:   "The Event Server for Control Plane",
	Version: "0.0.1",
	RunE:    run,
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.Flags().String("orgId", "", "The org ID for watching the resource")
	RootCmd.Flags().String("host", "127.0.0.1", "The Service GRPC Host to connect")
	RootCmd.Flags().Uint32("port", 8000, "The Service GRPC port to connect")
	RootCmd.MarkFlagRequired("orgId")
	RootCmd.Flags().String("topicSelfLink", "", "The self link of the WatchTopic")

	RootCmd.Flags().String("auth.privateKey", "./private_key.pem", "The private key associated with service account(default : ./private_key.pem)")
	RootCmd.Flags().String("auth.publicKey", "./public_key.pem", "The public key associated with service account(default : ./public_key.pem)")
	RootCmd.Flags().String("auth.keyPassword", "", "The password for private key")
	RootCmd.Flags().String("auth.URL", "https://login-preprod.axway.com/auth", "The AxwayID auth URL")
	RootCmd.Flags().String("auth.clientId", "", "The service account client ID")
	RootCmd.Flags().Duration("auth.timeout", 10*time.Second, "The connection timeout for AxwayID")
	RootCmd.MarkFlagRequired("auth.clientId")

	RootCmd.Flags().Bool("insecure", false, "Do not verify the server cert on TLS connection")
	RootCmd.Flags().String("log.level", "info", "log level")
	RootCmd.Flags().String("log.format", "json", "line or json")

	bindOrPanic("orgId", RootCmd.Flags().Lookup("orgId"))
	bindOrPanic("host", RootCmd.Flags().Lookup("host"))
	bindOrPanic("port", RootCmd.Flags().Lookup("port"))
	bindOrPanic("topicSelfLink", RootCmd.Flags().Lookup("topicSelfLink"))
	RootCmd.MarkFlagRequired("topicSelfLink")

	bindOrPanic("auth.privateKey", RootCmd.Flags().Lookup("auth.privateKey"))
	bindOrPanic("auth.publicKey", RootCmd.Flags().Lookup("auth.publicKey"))
	bindOrPanic("auth.keyPassword", RootCmd.Flags().Lookup("auth.keyPassword"))
	bindOrPanic("auth.URL", RootCmd.Flags().Lookup("auth.URL"))
	bindOrPanic("auth.clientId", RootCmd.Flags().Lookup("auth.clientId"))
	bindOrPanic("auth.timeout", RootCmd.Flags().Lookup("auth.timeout"))

	bindOrPanic("insecure", RootCmd.Flags().Lookup("insecure"))
	bindOrPanic("log.level", RootCmd.Flags().Lookup("log.level"))
	bindOrPanic("log.format", RootCmd.Flags().Lookup("log.format"))
}

func initConfig() {
	viper.SetTypeByDefaultValue(true)
	viper.SetEnvPrefix("watchclient")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

func configFromViper() client.Config {
	return client.Config{
		Host:          viper.GetString("host"),
		Port:          viper.GetUint32("port"),
		TenantID:      viper.GetString("orgId"),
		TopicSelfLink: viper.GetString("topicSelfLink"),
		Insecure:      viper.GetBool("insecure"),
		Auth: client.AuthConfig{
			PrivateKey:  viper.GetString("auth.privateKey"),
			PublicKey:   viper.GetString("auth.publicKey"),
			KeyPassword: viper.GetString("auth.keyPassword"),
			URL:         viper.GetString("auth.URL"),
			ClientID:    viper.GetString("auth.clientId"),
			Timeout:     viper.GetDuration("auth.timeout"),
		},
	}
}

func bindOrPanic(key string, flag *flag.Flag) {
	if err := viper.BindPFlag(key, flag); err != nil {
		panic(err)
	}
}

func run(cmd *cobra.Command, args []string) error {
	logger, err := getLogger(viper.GetString("log.level"), viper.GetString("log.format"))
	if err != nil {
		return err
	}
	log = logger.WithField("package", "cmd")

	config := configFromViper()
	log.Debugf("Config: %+v", config)
	wc, err := client.NewWatchClient(&config, logger)
	if err != nil {
		return err
	}
	wc.Watch()
	return nil
}
