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

func (c *cacheManager) GetTeamByName(name string) *defs.PlatformTeam {
	team := c.getTeamByName(name)
	if team == nil && c.teamRefreshHandler != nil {
		c.teamRefreshHandler()
		return c.getTeamByName(name)
	}
	return nil
}

// GetTeamByName gets a team by name
func (c *cacheManager) getTeamByName(name string) *defs.PlatformTeam {
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

func (c *cacheManager) GetTeamByID(id string) *defs.PlatformTeam {
	team := c.getTeamByID(id)
	if team == nil && c.teamRefreshHandler != nil {
		c.teamRefreshHandler()
		return c.getTeamByID(id)
	}
	return nil
}

// GetTeamByID gets a team by id
func (c *cacheManager) getTeamByID(id string) *defs.PlatformTeam {
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

// DeleteAccessControlList removes the Access Control List to the cache
func (c *cacheManager) DeleteAccessControlList() error {
	return c.teams.Delete(accessControlList)
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
