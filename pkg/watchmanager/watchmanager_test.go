package watchmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWatchmanager(t *testing.T) {
	wm, err := New("localhost", 8080, "tenantID", func() (string, error) {
		return "abc", nil
	})
	assert.Nil(t, err)
	assert.NotNil(t, wm)
	// ch := make(chan *proto.Event)
	// errCh := make(chan error)
	// subscriptionID, err := wm.RegisterWatch("selfLink", ch, errCh)
	// assert.Nil(t, err)
	// assert.NotNil(t, subscriptionID)
}
