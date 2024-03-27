package exception

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExceptionNoBlock(t *testing.T) {
	assert.Equal(t, 0, 0)
	noopException := Block{}.Do
	assert.NotPanics(t, noopException)

	noCatchBlockException := Block{
		Try: func() {
			Throw(errors.New("some_err"))
		},
	}.Do
	assert.Panics(t, noCatchBlockException)
}

func TestExceptionTryCatchBlock(t *testing.T) {
	var catchErr error
	tryCatchBlockException := Block{
		Try: func() {
			Throw(errors.New("try_catch_err"))
		},
		Catch: func(err error) {
			catchErr = err
		},
	}.Do

	assert.NotPanics(t, tryCatchBlockException)
	assert.NotNil(t, catchErr)
	assert.Equal(t, "try_catch_err", catchErr.Error())
}

func TestExceptionTryCatchFinallyBlock(t *testing.T) {
	var catchErr error
	var executionOrder string
	tryCatchFinallyBlockException := Block{
		Try: func() {
			executionOrder = "try"
			Throw(errors.New("try_catch_finally_err"))
		},
		Catch: func(err error) {
			executionOrder = executionOrder + "catch"
			catchErr = err
		},
		Finally: func() {
			executionOrder = executionOrder + "finally"
		},
	}.Do

	assert.NotPanics(t, tryCatchFinallyBlockException)
	assert.NotNil(t, catchErr)
	assert.Equal(t, "try_catch_finally_err", catchErr.Error())
	assert.Equal(t, "trycatchfinally", executionOrder)
}

func TestExceptionTryCatchBlockWithNPE(t *testing.T) {
	var catchErr error
	var executionOrder string
	tryCatchFinallyBlockException := Block{
		Try: func() {
			executionOrder = "try"
			b := Block{}
			b.Try()
		},
		Catch: func(err error) {
			executionOrder = executionOrder + "catch"
			catchErr = err
		},
	}.Do

	assert.NotPanics(t, tryCatchFinallyBlockException)
	assert.NotNil(t, catchErr)
	assert.Equal(t, strings.Contains(catchErr.Error(), "runtime error: invalid memory address or nil pointer dereference"), true)
}
