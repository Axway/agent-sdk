package awsconfig

import (
	"errors"
	"time"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/exception"
)

// AWSConfig - Interface for aws config
type AWSConfig interface {
	GetPollInterval() time.Duration
	GetRegion() string
	GetQueueName() string
	GetLogGroupArn() string
	GetStageTags() string
	GetAccessKey() string
	GetSecretKey() string
	validate()
}

// AWSConfiguration - AWS Configuration
type AWSConfiguration struct {
	PollInterval time.Duration
	Region       string
	QueueName    string
	LogGroupArn  string
	StageTags    string
	AccessKey    string
	SecretKey    string
}

// NewAWSConfig - Creates the default aws config
func NewAWSConfig() AWSConfig {
	return &AWSConfiguration{
		PollInterval: 30 * time.Second,
	}
}

// Validate - Validates the config
func (a *AWSConfiguration) Validate() (err error) {
	exception.Block{
		Try: func() {
			a.validate()
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()

	return
}

func (a *AWSConfiguration) validate() {
	if a.GetRegion() == "" {
		exception.Throw(errors.New("Error aws.region not set in config"))
	}

	if a.GetQueueName() == "" {
		exception.Throw(errors.New("Error aws.queueName not set in config"))
	}

	if a.GetAccessKey() == "" {
		exception.Throw(errors.New("Error aws.auth.accessKey not set in config"))
	}

	if a.GetSecretKey() == "" {
		exception.Throw(errors.New("Error aws.auth.secretKey not set in config"))
	}

	return
}

// GetPollInterval - Returns the poll Interval
func (a *AWSConfiguration) GetPollInterval() time.Duration {
	return a.PollInterval
}

// GetRegion - Returns the AWS region
func (a *AWSConfiguration) GetRegion() string {
	return a.Region
}

// GetQueueName - Returns the AWS SQS queue name
func (a *AWSConfiguration) GetQueueName() string {
	return a.QueueName
}

// GetLogGroupArn - Returns the AWS Log Group Arn
func (a *AWSConfiguration) GetLogGroupArn() string {
	return a.LogGroupArn
}

// GetStageTags - Returns the stage tags
func (a *AWSConfiguration) GetStageTags() string {
	return a.StageTags
}

// GetAccessKey - Returns the AWS access key
func (a *AWSConfiguration) GetAccessKey() string {
	return a.AccessKey
}

// GetSecretKey - Returns the AWS secret key
func (a *AWSConfiguration) GetSecretKey() string {
	return a.SecretKey
}

