package agent

import (
	"reflect"

	"github.com/Axway/agent-sdk/pkg/config"
)

// ApplyResourceToConfig - applies the resources to agent configs
// Uses reflection to get the IResourceConfigCallback interface on the config struct or
// struct variable.
// Makes call to ApplyResources method with dataplane and agent resources from API server
func ApplyResourceToConfig(cfg interface{}) error {
	// This defer func is to catch a possible panic that WILL occur if the cfg object that is passed in embedds the IResourceConfig interface
	// within its struct, but does NOT implement the ApplyResources method. While it might be that this method really isn't necessary, we will
	// log an error alerting the user in case it wasn't intentional.
	// defer util.HandleInterfaceFuncNotImplemented(cfg, "ApplyResources", "IResourceConfigCallback")

	agentRes := GetAgentResource()
	if agentRes == nil {
		return nil
	}

	if objInterface, ok := cfg.(config.IResourceConfigCallback); ok {
		err := objInterface.ApplyResources(agentRes)
		if err != nil {
			return err
		}
	}

	// If the parameter is of struct pointer, use indirection to get the
	// real value object
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}

	// Look for ApplyResouceToConfig method on struct properties and invoke it
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() {
			fieldInterface := v.Field(i).Interface()
			if objInterface, ok := fieldInterface.(config.IResourceConfigCallback); ok {
				err := ApplyResourceToConfig(objInterface)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
