package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoffTimeout(t *testing.T) {
	start := time.Millisecond
	max := 10 * time.Millisecond
	factor := 2
	newBT := newBackoffTimeout(start, max, factor)

	// Verify initial values using getter methods
	assert.Equal(t, start, newBT.getBaseTimeout())
	assert.Equal(t, max, newBT.getMaxTimeout())
	assert.Equal(t, factor, newBT.factor)
	assert.Equal(t, start, newBT.getCurrentTimeout())

	// Increase the timeout, should double the current timeout
	newBT.increaseTimeout()
	assert.Equal(t, start, newBT.getBaseTimeout())
	assert.Equal(t, max, newBT.getMaxTimeout())
	assert.Equal(t, factor, newBT.factor)
	assert.Equal(t, start*2, newBT.getCurrentTimeout())

	// Increase the timeout again, should double again
	newBT.increaseTimeout()
	assert.Equal(t, start*2*2, newBT.getCurrentTimeout())

	// Increase the timeout 2 more times to exceed max, should reset to base
	newBT.increaseTimeout()
	newBT.increaseTimeout()
	assert.Equal(t, start, newBT.getCurrentTimeout())

	// Reset the timeout, should set current timeout to base
	newBT.reset()
	assert.Equal(t, start, newBT.getCurrentTimeout())

	// Call sleep, ensure no changes to timeouts
	newBT.sleep()
	assert.Equal(t, start, newBT.getBaseTimeout())
	assert.Equal(t, max, newBT.getMaxTimeout())
	assert.Equal(t, factor, newBT.factor)
	assert.Equal(t, start, newBT.getCurrentTimeout())
}
