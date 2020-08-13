package notify

import agenterrors "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"

// Errors hit when sending subscription notifications
var (
	ErrSubscriptionNotification        = agenterrors.Newf(1300, "could not send notification via %s, check SUBSCRIPTION config")
	ErrSubscriptionNoNotifications     = agenterrors.New(1301, "no subscription notification type is configured, check SUBSCRIPTION config")
	ErrSubscriptionData                = agenterrors.New(1302, "error creating notification request")
	ErrSubscriptionBadAuthtype         = agenterrors.Newf(1303, "email template not updated because an invalid authType was supplied: %s. Check subscriptions.smtp.authType")
	ErrSubscriptionNoTemplateForAction = agenterrors.Newf(1304, "no email template found for action %s")
	ErrSubscriptionSendEmail           = agenterrors.New(1305, "error sending email to SMTP server")
)
