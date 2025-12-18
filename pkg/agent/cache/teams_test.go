package cache

import (
	"testing"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestTeamCache(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	team1 := &defs.PlatformTeam{
		ID:      "id1",
		Name:    "name1",
		Default: false,
	}
	team2 := &defs.PlatformTeam{
		ID:      "id2",
		Name:    "name2",
		Default: true,
	}

	m.AddTeam(team1)
	m.AddTeam(team2)

	cachedTeam := m.GetTeamByName("name1")
	assert.Equal(t, team1, cachedTeam)

	cachedTeam = m.GetTeamByID("id1")
	assert.Equal(t, team1, cachedTeam)

	cachedTeam = m.GetDefaultTeam()
	assert.Equal(t, team2, cachedTeam)

	acl := createRequestDefinition("name1", "id1")

	m.SetAccessControlList(acl)

	cachedACL := m.GetAccessControlList()
	assert.Equal(t, acl, cachedACL)

	err := m.DeleteAccessControlList()
	assert.Nil(t, err)

	cachedACL = m.GetAccessControlList()
	assert.Nil(t, cachedACL)
}
