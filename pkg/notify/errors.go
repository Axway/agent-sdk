package notify

import agenterrors "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/errors"

// Errors hit when sending subscription notifications
var (
	ErrSubscriptionNotification    = agenterrors.Newf(1300, "could not send notification via %s, check SUBSCRIPTION config")
	ErrSubscriptionNoNotifications = agenterrors.New(1301, "no subscription notification type is configured, check SUBSCRIPTION config")
	ErrSubscriptionData            = agenterrors.New(1302, "error creating notification request")
)
