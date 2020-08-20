package config

import configerrors "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"

// Errors hit when sending subscription notifications
var (
	ErrSubscriptionApprovalModeInvalid = configerrors.New(1301, "error central.subscriptions.approvalmode set to incorrect value in config")
)
