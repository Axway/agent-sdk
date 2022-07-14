package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrationConfig(t *testing.T) {
	defConf := newMigrationConfig()

	assert.False(t, defConf.ShouldCleanInstances())
}
