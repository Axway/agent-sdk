package watchmanager

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/config"
)

func TestWatchmanager(t *testing.T) {
	centralCfg := &config.CentralConfiguration{}
	wm := New(centralCfg, func() (string, error) {
		return "abc", nil
	})
	cfg := Config{
		ScopeKind:  "",
		Scope:      "",
		Group:      "management",
		Kind:       "Environment",
		Name:       "abc",
		EventTypes: []string{"CREATED", "UPDATED", "DELETED"},
	}
	ch := make(chan *proto.Event)
	ctx, err := wm.RegisterWatch(cfg, ch)
	fmt.Println(ctx)
	fmt.Println(err)
}
