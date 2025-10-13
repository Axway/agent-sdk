package exception

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Block - defines the try, catch and finally code blocks
type Block struct {
	Try     func()
	Catch   func(error)
	Finally func()
}

// Throw - raises the error
// Raises the panic error
func Throw(err error) {
	panic(err)
}

// Do - Executes the Exception block.
// 1. Defers the finally method, so that it can be called last
// 2. Defers the execution of catch, to recover from panic raised by Throw and callback Catch method
// 3. Executes the try method, that may raise error by calling Throw method
func (block Block) Do() {
	if block.Try == nil {
		return
	}

	if block.Finally != nil {
		defer block.Finally()
	}

	if block.Catch != nil {
		defer func() {
			if obj := recover(); obj != nil {
				err := obj.(error)
				if strings.Contains(err.Error(), "nil pointer dereference") {
					block.Catch(fmt.Errorf("%s: \n err.Error() %s", err.Error(), string(debug.Stack())))
				} else {
					block.Catch(err)
				}
			}
		}()
	}
	block.Try()
}
