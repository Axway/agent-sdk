package apic

import (
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
)

func addSpecHashToResource(h v1.Interface) error {
	ri, err := h.AsInstance()
	if err != nil {
		return err
	}

	hashInt, err := util.ComputeHash(ri.Spec)
	if err != nil {
		return err
	}

	util.SetAgentDetailsKey(h, definitions.AttrSpecHash, fmt.Sprintf("%v", hashInt))
	return nil
}
