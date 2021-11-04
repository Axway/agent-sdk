package v1

import (
	"encoding/json"
	"fmt"
)

// OwnerType -
type OwnerType uint

// values for Owner.Team
const (
	TeamOwner OwnerType = iota
)

// map of ownertype to string
var ownerTypeToString = map[OwnerType]string{
	TeamOwner: "team",
}

var ownerTypeFromString = map[string]OwnerType{
	"team": TeamOwner,
}

// Owner structure.
type Owner struct {
	Type OwnerType `json:"type,omitempty"`
	ID   string    `json:"id"`
}

// SetType -
func (o *Owner) SetType(t OwnerType) {
	o.Type = t
}

// SetID -
func (o *Owner) SetID(id string) {
	o.ID = id
}

// MarshalJSON -
func (o *Owner) MarshalJSON() ([]byte, error) {
	var t string
	var ok bool
	if t, ok = ownerTypeToString[o.Type]; !ok {
		return nil, fmt.Errorf("unknown owner type %d", o.Type)
	}

	type Alias Owner
	return json.Marshal(&struct {
		*Alias
		Type string `json:"type,omitempty"`
	}{
		Alias: (*Alias)(o),
		Type:  t,
	})
}

// UnmarshalJSON -
func (o *Owner) UnmarshalJSON(bytes []byte) error {
	type Alias Owner
	aux := struct {
		*Alias
		Type string `json:"type,omitempty"`
	}{
		Alias: (*Alias)(o),
	}

	if err := json.Unmarshal(bytes, &aux); err != nil {
		return err
	}

	var t OwnerType
	var ok bool
	if t, ok = ownerTypeFromString[aux.Type]; !ok {
		return fmt.Errorf("unknown owner type %d", o.Type)
	}
	o.Type = t
	return nil
}
