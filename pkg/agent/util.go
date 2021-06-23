package agent

import (
	"reflect"
	"time"

	v1Time "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
)

// getTimestamp - Returns current timestamp formatted for API Server
// if the local status exists, return the local timestamp, otherwise return Now()
func getTimestamp() v1Time.Time {
	activityTime := time.Now()
	if statusUpdate != nil {
		curTime := getLocalActivityTime()
		if !curTime.IsZero() {
			activityTime = curTime
		}
	}
	newV1Time := v1Time.Time(activityTime)
	return newV1Time
}

// ApplyResouceToConfig - applies the resources to agent configs
// Uses reflection to get the IResourceConfigCallback interface on the config struct or
// struct variable.
// Makes call to ApplyResources method with dataplane and agent resources from API server
func ApplyResouceToConfig(cfg interface{}) error {
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
