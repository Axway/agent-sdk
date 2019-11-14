package awsconfig

import (
	"testing"
	"time"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/exception"
	"github.com/stretchr/testify/assert"
)

func validateAWS(cfg AWSConfig) (err error) {
	exception.Block{
		Try: func() {
			cfg.validate()
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()
	return
}

func TestAWSConfig(t *testing.T) {
	cfg := NewAWSConfig()
	awsCfg := cfg.(*AWSConfiguration)
	err := awsCfg.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, "Error aws.region not set in config", err.Error())
	assert.Equal(t, 30*time.Second, cfg.GetPollInterval())

	region := "eu-west-2"
	awsCfg.Region = region
	err = awsCfg.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, "Error aws.queueName not set in config", err.Error())
	assert.Equal(t, region, cfg.GetRegion())

	queue := "queue"
	awsCfg.QueueName = queue
	err = awsCfg.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, "Error aws.auth.accessKey not set in config", err.Error())
	assert.Equal(t, queue, cfg.GetQueueName())

	accessKey := "cccc"
	awsCfg.AccessKey = accessKey
	err = awsCfg.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, "Error aws.auth.secretKey not set in config", err.Error())
	assert.Equal(t, accessKey, cfg.GetAccessKey())

	secretKey := "ppp"
	awsCfg.SecretKey = secretKey
	err = awsCfg.Validate()
	assert.Nil(t, err)
	assert.Equal(t, secretKey, cfg.GetSecretKey())
	assert.Equal(t, "", cfg.GetLogGroupArn())
	assert.Equal(t, "", cfg.GetStageTags())
}
