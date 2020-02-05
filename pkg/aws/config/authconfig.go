package config

import (
	"errors"

	"git.ecd.axway.int/apigov/apic_agents_core/pkg/exception"
)

// AWSAuthConfig - Interface for aws authentication config
type AWSAuthConfig interface {
	GetAccessKey() string
	GetSecretKey() string
	validate()
}

// AWSAuthConfiguration - AWS Authentication Configuration
type AWSAuthConfiguration struct {
	AccessKey string `config:"accessKey"`
	SecretKey string `config:"secretKey"`
}

func (a *AWSAuthConfiguration) validate() {
	if a.GetAccessKey() == "" {
		exception.Throw(errors.New("Error aws.auth.accessKey not set in config"))
	}

	if a.GetSecretKey() == "" {
		exception.Throw(errors.New("Error aws.auth.secretKey not set in config"))
	}

	return
}

// GetAccessKey - Returns the AWS access key
func (a *AWSAuthConfiguration) GetAccessKey() string {
	return a.AccessKey
}

// GetSecretKey - Returns the AWS secret key
func (a *AWSAuthConfiguration) GetSecretKey() string {
	return a.SecretKey
}
