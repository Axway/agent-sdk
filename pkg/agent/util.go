package agent

import (
	"reflect"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
)

// apiServerTimeFormat is the date-time format for API Server
const apiServerTimeFormat = "2006-01-02T03:04:05.000-0700"

// getTimestamp - Returns current timestamp formatted for API Server
func getTimestamp() string {
	return time.Now().Format(apiServerTimeFormat)
}

// ApplyResouceToConfig - applies the resources to agent configs
// Uses reflection to get the IResourceConfigCallback interface on the config struct or
// struct variable.
// Makes call to ApplyResources method with dataplane and agent resources from API server
func ApplyResouceToConfig(cfg interface{}) error {
	dp := GetDataplaneResource()
	agentRes := GetAgentResource()
	if dp == nil || agentRes == nil {
		return nil
	}

	if objInterface, ok := cfg.(config.IResourceConfigCallback); ok {
		err := objInterface.ApplyResources(dp, agentRes)
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

	// Look for Validate method on struct properties and invoke it
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() {
			fieldInterface := v.Field(i).Interface()
			if objInterface, ok := fieldInterface.(config.IResourceConfigCallback); ok {
				err := ApplyResouceToConfig(objInterface)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
