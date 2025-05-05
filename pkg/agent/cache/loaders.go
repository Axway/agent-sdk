package cache

import (
	"encoding/json"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/cache"
)

type resourceLoader struct {
	setter func(cache.Cache, string)
	key    string
}

func createResourceLoader(setter func(cache.Cache, string), key string) *resourceLoader {
	return &resourceLoader{
		setter: setter,
		key:    key,
	}
}

func (rl *resourceLoader) getkey() string {
	return rl.key
}

func (rl *resourceLoader) loaded(c cache.Cache) {
	rl.setter(c, rl.key)
}

func (resourceLoader) unmarshaller(data []byte) (interface{}, error) {
	ri := &v1.ResourceInstance{}
	err := json.Unmarshal(data, ri)
	if err != nil {
		return nil, err
	}
	ri.CreateHashes()
	return ri, nil
}

func createInstanceCountLoader(setter func(cache.Cache, string), key string) *instanceCountLoader {
	return &instanceCountLoader{createResourceLoader(setter, key)}
}

// instanceCountLoader
type instanceCountLoader struct {
	*resourceLoader
}

func (l *instanceCountLoader) unmarshaller(data []byte) (interface{}, error) {
	c := apiServiceToInstanceCount{}
	err := json.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func createSequenceLoader(setter func(cache.Cache, string), key string) *sequenceLoader {
	return &sequenceLoader{createResourceLoader(setter, key)}
}

type sequenceLoader struct {
	*resourceLoader
}

func (sequenceLoader) unmarshaller(data []byte) (interface{}, error) {
	seq := int64(0)
	err := json.Unmarshal(data, &seq)
	if err != nil {
		return nil, err
	}
	return seq, nil
}

func createTeamLoader(setter func(cache.Cache, string), key string) *teamLoader {
	return &teamLoader{createResourceLoader(setter, key)}
}

type teamLoader struct {
	*resourceLoader
}

func (teamLoader) unmarshaller(data []byte) (interface{}, error) {
	team := &defs.PlatformTeam{}
	err := json.Unmarshal(data, team)
	if err != nil {
		return nil, err
	}
	return team, nil
}
