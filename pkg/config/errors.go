package config

import configerrors "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"

// Errors hit when validating or parsing config
var (
	ErrSubscriptionApprovalModeInvalid = configerrors.New(1401, "error central.subscriptions.approvalmode set to an incorrect value in config")
)
