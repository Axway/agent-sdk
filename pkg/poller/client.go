package poller

import (
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/harvester"
)

func NewPollClient(cfg config.CentralConfig, getToken auth.TokenGetter, sq harvester.SequenceProvider) {
	tls := cfg.GetTLSConfig()
	hcfg := &harvester.Config{
		ClientTimeout:    cfg.GetClientTimeout(),
		Host:             cfg.GetURL(),
		PageSize:         100,
		Port:             443,
		Protocol:         "",
		ProxyURL:         cfg.GetProxyURL(),
		SequenceProvider: sq,
		TenantID:         cfg.GetTenantID(),
		TlsCfg:           cfg.GetTLSConfig().BuildTLSConfig(),
		TokenGetter:      getToken.GetToken,
	}
}
