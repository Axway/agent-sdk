package agent

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	v1Time "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
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

// ApplyResourceToConfig - applies the resources to agent configs
// Uses reflection to get the IResourceConfigCallback interface on the config struct or
// struct variable.
// Makes call to ApplyResources method with dataplane and agent resources from API server
func ApplyResourceToConfig(cfg interface{}) error {
	obj := cfg
	defer func() {
		if err := recover(); err != nil {
			str := fmt.Sprintf("%v", err)
			if strings.Contains(str, "nil pointer dereference") {
				log.Errorf("The function 'ApplyResources' for interface IResourceConfigCallback is not implemented in %s.", reflect.TypeOf(obj))
			}
		}
		// }
	}()

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
