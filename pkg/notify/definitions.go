package notify

import (
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
)

var globalCfg corecfg.SubscriptionConfig
var templateActionMap map[apic.SubscriptionState]*config.EmailTemplate

// SetSubscriptionConfig - Creates the default subscription config
func SetSubscriptionConfig(cfg corecfg.SubscriptionConfig) {
	globalCfg = cfg

	templateActionMap = map[apic.SubscriptionState]*config.EmailTemplate{
		apic.SubscriptionActive:              cfg.GetSubscribeTemplate(),
		apic.SubscriptionUnsubscribed:        cfg.GetUnsubscribeTemplate(),
		apic.SubscriptionFailedToSubscribe:   cfg.GetSubscribeFailedTemplate(),
		apic.SubscriptionFailedToUnsubscribe: cfg.GetUnsubscribeFailedTemplate(),
	}
}
