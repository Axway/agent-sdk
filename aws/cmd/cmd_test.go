package cmd

import (
	"fmt"
	"testing"
	"time"

	corecmd "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/cmd"
	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func getPFlag(cmd corecmd.AgentRootCmd, flagName string) *flag.Flag {
	return cmd.RootCmd().Flags().Lookup(flagName)
}

func assertCmdFlag(t *testing.T, cmd corecmd.AgentRootCmd, flagName, fType, description string) {
	pflag := getPFlag(cmd, flagName)
	assert.NotNil(t, pflag)
	assert.Equal(t, fType, pflag.Value.Type())
	assert.Equal(t, description, pflag.Usage)
}

func assertStringCmdFlag(t *testing.T, cmd corecmd.AgentRootCmd, propertyName, flagName, defaultVal, description string) {
	assertCmdFlag(t, cmd, flagName, "string", description)
	assert.Equal(t, defaultVal, viper.GetString(propertyName))
}

func assertBoolCmdFlag(t *testing.T, cmd corecmd.AgentRootCmd, propertyName, flagName string, defaultVal bool, description string) {
	assertCmdFlag(t, cmd, flagName, "bool", description)
	assert.Equal(t, defaultVal, viper.GetString(propertyName))
}

func assertDurationCmdFlag(t *testing.T, cmd corecmd.AgentRootCmd, propertyName, flagName string, defaultVal time.Duration, description string) {
	assertCmdFlag(t, cmd, flagName, "duration", description)
	assert.Equal(t, defaultVal, viper.GetDuration(propertyName))
}

func TestAWSCmdFlags(t *testing.T) {
	// Discovery Agent
	rootCmd := corecmd.NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.DiscoveryAgent)
	AddAWSProperties(rootCmd)
	assertDurationCmdFlag(t, rootCmd, "aws.pollInterval", "awsPollInterval", 20*time.Second, "Interval between polling the SQS Queue")
	assertStringCmdFlag(t, rootCmd, "aws.queueName", "awsQueueName", "", "SQS Queue Name that we are polling")
	assertStringCmdFlag(t, rootCmd, "aws.region", "awsRegion", "", "AWS Region that we are watching for changes")
	assertStringCmdFlag(t, rootCmd, "aws.auth.accessKey", "awsAccessKey", "", "Access Key for AWS Authentication")
	assertStringCmdFlag(t, rootCmd, "aws.auth.secretKey", "awsSecretKey", "", "Secret Key for AWS Authentication")
	assertStringCmdFlag(t, rootCmd, "aws.logGroupArn", "awsLogGroupArn", "", "AWS Log Group ARN for AWS APIGW Access logs")
	assertStringCmdFlag(t, rootCmd, "aws.discoveryTags", "awsDiscoveryTags", "PublishToCentral", "Tags on AWS APIGW stages that will be discovered by the agent")
	assertBoolCmdFlag(t, rootCmd, "aws.pushTags", "awsPushTags", true, "Push the Tags on AWS APIGW stages to API Central")

	// Traceability Agent
	rootCmd = corecmd.NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.TraceabilityAgent)
	AddAWSProperties(rootCmd)
	assertDurationCmdFlag(t, rootCmd, "aws.pollInterval", "awsPollInterval", 20*time.Second, "Interval between polling the SQS Queue")
	assertStringCmdFlag(t, rootCmd, "aws.queueName", "awsQueueName", "", "SQS Queue Name that we are polling")
	assertStringCmdFlag(t, rootCmd, "aws.region", "awsRegion", "", "AWS Region that we are watching for changes")
	assertStringCmdFlag(t, rootCmd, "aws.auth.accessKey", "awsAccessKey", "", "Access Key for AWS Authentication")
	assertStringCmdFlag(t, rootCmd, "aws.auth.secretKey", "awsSecretKey", "", "Secret Key for AWS Authentication")
}

func TestAWSCmdConfigDefault(t *testing.T) {
	// Discovery
	rootCmd := corecmd.NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.DiscoveryAgent)
	AddAWSProperties(rootCmd)
	rootCmd.RootCmd().SetArgs([]string{
		fmt.Sprintf("--awsRegion=%s", "eu-west-1"),
		fmt.Sprintf("--awsQueueName=%s", "queue"),
		fmt.Sprintf("--awsAccessKey=%s", "123"),
		fmt.Sprintf("--awsSecretKey=%s", "346"),
		fmt.Sprintf("--awsLogGroupArn=%s", "/"),
	})
	fExecute := func() {
		rootCmd.Execute()
	}
	// This panics because we are not setting all of the central config in the test
	assert.Panics(t, fExecute)

	awsConfig, err := ParseAWSConfig(rootCmd)

	assert.Nil(t, err, "Parsing AWS Config returned error")
	assert.Equal(t, 20*time.Second, awsConfig.GetPollInterval())
	assert.Equal(t, "PublishToCentral", awsConfig.GetDiscoveryTags())
	assert.Equal(t, true, awsConfig.ShouldPushTags())
	assert.Equal(t, "eu-west-1", awsConfig.GetRegion())
	assert.Equal(t, "queue", awsConfig.GetQueueName())
	assert.Equal(t, "123", awsConfig.GetAuthConfig().GetAccessKey())
	assert.Equal(t, "346", awsConfig.GetAuthConfig().GetSecretKey())
	assert.Equal(t, "/", awsConfig.GetLogGroupArn())

	// Traceability
	rootCmd = corecmd.NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.TraceabilityAgent)
	AddAWSProperties(rootCmd)
	rootCmd.RootCmd().SetArgs([]string{
		fmt.Sprintf("--awsRegion=%s", "eu-west-1"),
		fmt.Sprintf("--awsQueueName=%s", "queue"),
		fmt.Sprintf("--awsAccessKey=%s", "123"),
		fmt.Sprintf("--awsSecretKey=%s", "346"),
	})
	// This panics because we are not setting all of the central config in the test
	assert.Panics(t, fExecute)

	awsConfig, err = ParseAWSConfig(rootCmd)

	assert.Nil(t, err, "Parsing AWS Config returned error")
	assert.Equal(t, 20*time.Second, awsConfig.GetPollInterval())
	assert.Equal(t, "eu-west-1", awsConfig.GetRegion())
	assert.Equal(t, "queue", awsConfig.GetQueueName())
	assert.Equal(t, "123", awsConfig.GetAuthConfig().GetAccessKey())
	assert.Equal(t, "346", awsConfig.GetAuthConfig().GetSecretKey())
}
