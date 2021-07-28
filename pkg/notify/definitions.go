package notify

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

var globalCfg corecfg.SubscriptionConfig
var templateActionMap map[apic.SubscriptionState]*corecfg.EmailTemplate

// SetSubscriptionConfig - Creates the default subscription config
func SetSubscriptionConfig(cfg corecfg.SubscriptionConfig) {
	globalCfg = cfg

	templateActionMap = map[apic.SubscriptionState]*corecfg.EmailTemplate{
		apic.SubscriptionActive:              cfg.GetSubscribeTemplate(),
		apic.SubscriptionUnsubscribed:        cfg.GetUnsubscribeTemplate(),
		apic.SubscriptionFailedToSubscribe:   cfg.GetSubscribeFailedTemplate(),
		apic.SubscriptionFailedToUnsubscribe: cfg.GetUnsubscribeFailedTemplate(),
	}
}
