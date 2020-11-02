package config

import configerrors "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"

// Errors hit when validating or parsing config
var (
	ErrSubscriptionApprovalModeInvalid = configerrors.New(1401, "error central.subscriptions.approvalmode set to an incorrect value in config")
	ErrEnvConfigOverride               = configerrors.New(1402, "error in overriding configuration using environment variables")
	ErrStatusHealthCheckPeriod         = configerrors.New(1403, "invalid value for statusHealthCheckPeriod. Value must be between 1 and 5 minutes")
	ErrStatusHealthCheckInterval       = configerrors.New(1404, "invalid value for statusHealthCheckInterval. Value must be between 30 seconds and 5 minutes")
)
