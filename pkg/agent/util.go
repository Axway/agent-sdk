package agent

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
)

// ApplyResourceToConfig - applies the resources to agent configs
// Uses reflection to get the IResourceConfigCallback interface on the config struct or
// struct variable.
// Makes call to ApplyResources method with dataplane and agent resources from API server
func ApplyResourceToConfig(cfg interface{}) error {
	// This defer func is to catch a possible panic that WILL occur if the cfg object that is passed in embedds the IResourceConfig interface
	// within its struct, but does NOT implement the ApplyResources method. While it might be that this method really isn't necessary, we will
	// log an error alerting the user in case it wasn't intentional.
	defer util.HandleInterfaceFuncNotImplemented(cfg, "ApplyResources", "IResourceConfigCallback")

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

type atomicUint struct {
	num uint32
}

// value - returns the uint value within the atomicUint
func (a *atomicUint) value() uint32 {
	return atomic.LoadUint32(&a.num)
}

// increment - increases the uint value by 1
func (a *atomicUint) increment() uint32 {
	return atomic.AddUint32(&a.num, uint32(1))
}

// decrement - decreases the uint value by 1
func (a *atomicUint) decrement() uint32 {
	return atomic.AddUint32(&a.num, ^uint32(0))
}

// wait - waits for the uint to become 0, with no time limit
func (a *atomicUint) wait() {
	// waits for the int to be 0 before returning
	for {
		if atomic.LoadUint32(&a.num) == 0 {
			return
		}
	}
}

// waitMaxDuration - waits for the uint to become 0, up to duration
func (a *atomicUint) waitMaxDuration(duration time.Duration) error {
	// waits for the int to be 0 before returning
	maxDur := time.NewTicker(duration)
	defer maxDur.Stop()
	for {
		select {
		case <-maxDur.C:
			return fmt.Errorf("max duration hit while waiting")
		default:
			if atomic.LoadUint32(&a.num) == 0 {
				return nil
			}
		}
	}
}
