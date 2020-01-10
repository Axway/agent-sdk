package cmd

import (
	"time"

	awsconfig "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/aws/config"
	corecmd "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/cmd"
	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
)

//AddAWSProperties - Add AWS properties and command line flag
func AddAWSProperties(rootCmd corecmd.AgentRootCmd) {
	rootCmd.AddDurationProperty("aws.pollInterval", "awsPollInterval", 20*time.Second, "Interval between polling the SQS Queue")
	rootCmd.AddStringProperty("aws.region", "awsRegion", "", "AWS Region that we are watching for changes")
	rootCmd.AddStringProperty("aws.queueName", "awsQueueName", "", "SQS Queue Name that we are polling")
	rootCmd.AddStringProperty("aws.auth.accessKey", "awsAccessKey", "", "Access Key for AWS Authentication")
	rootCmd.AddStringProperty("aws.auth.secretKey", "awsSecretKey", "", "Secret Key for AWS Authentication")

	if rootCmd.GetAgentType() == corecfg.DiscoveryAgent {
		rootCmd.AddStringProperty("aws.logGroupArn", "awsLogGroupArn", "", "AWS Log Group ARN for AWS APIGW Access logs")
		rootCmd.AddBoolProperty("aws.pushTags", "awsPushTags", false, "Push the Tags on AWS APIGW stages to API Central")
		rootCmd.AddStringProperty("aws.filter", "awsFilter", "", "Filter condition for discovery")
	}
}

//ParseAWSConfig - Creates the AWSConfig object for the agent
func ParseAWSConfig(rootCmd corecmd.AgentRootCmd) (awsconfig.AWSConfig, error) {
	cfg := &awsconfig.AWSConfiguration{
		PollInterval: rootCmd.DurationPropertyValue("aws.pollInterval"),
		Region:       rootCmd.StringPropertyValue("aws.region"),
		QueueName:    rootCmd.StringPropertyValue("aws.queueName"),
		Auth: &awsconfig.AWSAuthConfiguration{
			AccessKey: rootCmd.StringPropertyValue("aws.auth.accessKey"),
			SecretKey: rootCmd.StringPropertyValue("aws.auth.secretKey"),
		},
	}

	if rootCmd.GetAgentType() == corecfg.DiscoveryAgent {
		cfg.LogGroupArn = rootCmd.StringPropertyValue("aws.logGroupArn")
		cfg.PushTags = rootCmd.BoolPropertyValue("aws.pushTags")
		cfg.Filter = rootCmd.StringPropertyValue("aws.filter")
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}
