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

	assert.Equal(t, start, newBT.base)
	assert.Equal(t, max, newBT.max)
	assert.Equal(t, factor, newBT.factor)
	assert.Equal(t, start, newBT.getCurrentTimeout())

	// increase the timeout, 2 milliseconds
	newBT.increaseTimeout()
	assert.Equal(t, start, newBT.base)
	assert.Equal(t, max, newBT.max)
	assert.Equal(t, factor, newBT.factor)
	assert.Equal(t, start*2, newBT.getCurrentTimeout())

	// increase the timeout again, 4 milliseconds
	newBT.increaseTimeout()
	assert.Equal(t, start*2*2, newBT.getCurrentTimeout())

	// increase the timeout 2 more times, should be at max
	newBT.increaseTimeout()
	newBT.increaseTimeout()
	assert.Equal(t, max, newBT.getCurrentTimeout())

	// reset the timeout
	newBT.reset()
	assert.Equal(t, start, newBT.getCurrentTimeout())

	// call sleep
	newBT.sleep()
	assert.Equal(t, start, newBT.base)
	assert.Equal(t, max, newBT.max)
	assert.Equal(t, factor, newBT.factor)
	assert.Equal(t, start, newBT.getCurrentTimeout())
}
