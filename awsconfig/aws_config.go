package awsconfig

import "time"

// AWSConfig - Interface for aws config
type AWSConfig interface {
	GetPollInterval() time.Duration
	GetRegion() string
	GetQueueName() string
	GetLogGroupArn() string
	GetStageTags() string
	GetAccessKey() string
	GetSecretKey() string
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

// GetTokenURL - Returns the poll Interval
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
