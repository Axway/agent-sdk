package config

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
	ShouldPushTags() bool
	GetAuthConfig() AWSAuthConfig
	GetFilter() string
	Validate() error
}

// AWSConfiguration - AWS Configuration
type AWSConfiguration struct {
	PollInterval time.Duration
	Region       string        `config:"region"`
	QueueName    string        `config:"queueName"`
	LogGroupArn  string        `config:"logGroupArn"`
	Filter       string        `config:"filter"`
	PushTags     bool          `config:"pushTags"`
	Auth         AWSAuthConfig `config:"auth"`
}

// NewAWSConfig - Creates the default aws config
func NewAWSConfig() AWSConfig {
	return &AWSConfiguration{
		PollInterval: 20 * time.Second,
		Auth:         &AWSAuthConfiguration{},
	}
}

// Validate - Validates the config
func (a *AWSConfiguration) Validate() (err error) {
	exception.Block{
		Try: func() {
			a.validate()
			a.Auth.validate()
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

// ShouldPushTags - should the tags be pushed?
func (a *AWSConfiguration) ShouldPushTags() bool {
	return a.PushTags
}

// GetAuthConfig - Returns the Auth Config
func (a *AWSConfiguration) GetAuthConfig() AWSAuthConfig {
	return a.Auth
}

// GetFilter - Returns the Filter condition config
func (a *AWSConfiguration) GetFilter() string {
	return a.Filter
}
