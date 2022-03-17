package cache

import (
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
)

const accessControlList = "AccessControlListKey"

// GetTeamCache - returns the team cache
func (c *cacheManager) GetTeamCache() cache.Cache {
	return c.teams
}

// AddTeam saves a team to the cache
func (c *cacheManager) AddTeam(team *defs.PlatformTeam) {
	defer c.setCacheUpdated(true)
	c.teams.SetWithSecondaryKey(team.Name, team.ID, *team)
}

// GetTeamByName gets a team by name
func (c *cacheManager) GetTeamByName(name string) *defs.PlatformTeam {
	item, err := c.teams.Get(name)
	if err != nil {
		return nil
	}
	team, ok := item.(defs.PlatformTeam)
	if !ok {
		return nil
	}
	return &team
}

// GetDefaultTeam gets the default team
func (c *cacheManager) GetDefaultTeam() *defs.PlatformTeam {
	names := c.teams.GetKeys()

	var defaultTeam defs.PlatformTeam
	for _, name := range names {
		item, _ := c.teams.Get(name)
		team, ok := item.(defs.PlatformTeam)
		if !ok {
			continue
		}

		if team.Default {
			defaultTeam = team
			break
		}

		continue
	}

	return &defaultTeam
}

// GetTeamByID gets a team by id
func (c *cacheManager) GetTeamByID(id string) *defs.PlatformTeam {
	item, err := c.teams.GetBySecondaryKey(id)
	if err != nil {
		return nil
	}
	team, ok := item.(defs.PlatformTeam)
	if !ok {
		return nil
	}
	return &team
}

// SetAccessControlList saves the Access Control List to the cache
func (c *cacheManager) SetAccessControlList(acl *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)
	c.teams.Set(accessControlList, acl)
}

// GetAccessControlList gets the Access Control List from the cache
func (c *cacheManager) GetAccessControlList() *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, _ := c.teams.Get(accessControlList)
	if item != nil {
		instance, ok := item.(*v1.ResourceInstance)
		if ok {
			return instance
		}
	}
	return nil
}
