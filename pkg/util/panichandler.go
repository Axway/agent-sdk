package util

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// HandleInterfaceFuncNotImplemented - use this function to recover from a panic due to an interface not being implemented
func HandleInterfaceFuncNotImplemented(obj interface{}, funcName, interfaceName string) {
	if interfaceObj := recover(); interfaceObj != nil {
		errStr := fmt.Sprintf("%v", interfaceObj)

		// an interface problem will contain this string...
		if strings.Contains(errStr, "nil pointer dereference") {
			log.Warnf("The function '%s' for interface '%s' is not implemented in %s.", funcName, interfaceName, reflect.TypeOf(obj).String())
		}
	}
}
