package v1

import (
	"fmt"

	apiv1 "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
)

type NotFound struct {
	apiv1.GroupKind
	Name  string
	Scope string
}

func (nf *NotFound) Error() string {
	return fmt.Sprintf("not found: group: %s, kind %s, name %s, scope %s", nf.Group, nf.Kind, nf.Name, nf.Scope)
}
