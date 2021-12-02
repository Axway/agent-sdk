package v1alpha1

// GENERATE: All of the code below was replaced after code gneration

import apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

// AccessControlListSpec struct for AccessControlListSpec
type AccessControlListSpec struct {
	Rules    []AccessRules `json:"rules,omitempty"`
	Subjects []apiv1.Owner `json:"subjects,omitempty"`
}

// AccessRules struct for AccessRules
type AccessRules struct {
	// Resource level at which access is being granted.
	Access []AccessLevelScope `json:"access,omitempty"`
}

// AccessLevelScope struct for AccessLevelScope
type AccessLevelScope struct {
	// Resource level at which access is being granted.
	Level string `json:"level,omitempty"`
}
