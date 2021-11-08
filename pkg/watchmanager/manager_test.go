package watchmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWatchmanager(t *testing.T) {
	cfg := &Config{
		Host:     "localhost",
		Port:     8080,
		TenantID: "tenantID",
		TokenGetter: func() (string, error) {
			return "abc", nil
		},
	}
	wm, err := New(cfg, nil)
	assert.Nil(t, err)
	assert.NotNil(t, wm)
	// ch := make(chan *proto.Event)
	// errCh := make(chan error)
	// subscriptionID, err := wm.RegisterWatch("selfLink", ch, errCh)
	// assert.Nil(t, err)
	// assert.NotNil(t, subscriptionID)
}
