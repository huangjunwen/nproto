package rpc

import (
	"context"
	"fmt"
	"reflect"
)

// RPCSpec is the contract between rpc server and client.
//
// It should be filled and call Validate() before use. After that don't modify its content.
type RPCSpec struct {
	// SvcName is the name of service.
	SvcName string

	// MethodName is the name of method.
	MethodName string

	// NewInput is used to generate a new input parameter. Must be a pointer.
	NewInput func() interface{}

	// NewOutput is used to generate a new output parameter. Must be a pointer.
	NewOutput func() interface{}

	inputType  reflect.Type
	outputType reflect.Type
}

// RPCHandler do the real job. RPCHandler can be client side or server side.
// It must be able to transfer normal error and nproto.Error from server side
// to client side.
type RPCHandler func(context.Context, interface{}) (interface{}, error)

// RPCMiddleware wraps a RPCHandler into another one.
type RPCMiddleware func(spec *RPCSpec, handler RPCHandler) RPCHandler

// Validate the RPCSpec. Call this before any other methods.
func (spec *RPCSpec) Validate() error {
	// Set inputType in the end of Validate.
	if spec.Validated() {
		return nil
	}

	if spec.SvcName == "" {
		return fmt.Errorf("RPCSpec.SvcName is empty")
	}
	if spec.MethodName == "" {
		return fmt.Errorf("RPCSpec.MethodName is empty")
	}
	if spec.NewInput == nil {
		return fmt.Errorf("RPCSpec.NewInput is empty")
	}
	if spec.NewOutput == nil {
		return fmt.Errorf("RPCSpec.NewOutput is empty")
	}

	newOutput := spec.NewOutput()
	if newOutput == nil {
		return fmt.Errorf("RPCSpec.NewOutput() returns nil")
	}
	outputType := reflect.TypeOf(newOutput)
	if outputType.Kind() != reflect.Ptr {
		return fmt.Errorf("RPCSpec.NewOutput() returns non-pointer")
	}

	newInput := spec.NewInput()
	if newInput == nil {
		return fmt.Errorf("RPCSpec.NewInput() returns nil")
	}
	inputType := reflect.TypeOf(newInput)
	if inputType.Kind() != reflect.Ptr {
		return fmt.Errorf("RPCSpec.NewInput() returns non-pointer")
	}

	spec.outputType = outputType
	spec.inputType = inputType
	return nil
}

// Validated returns true if Validate() has been called successful.
func (spec *RPCSpec) Validated() bool {
	return spec.inputType != nil
}

// AssertInputType makes sure input's type conform to the spec.
func (spec *RPCSpec) AssertInputType(input interface{}) error {
	if !spec.Validated() {
		return fmt.Errorf("RPCSpec has not validated yet")
	}
	if inputType := reflect.TypeOf(input); inputType != spec.inputType {
		return fmt.Errorf("%s input expect %s, but got %s", spec.String(), spec.inputType.String(), inputType.String())
	}
	return nil
}

// AssertOutputType makes sure output's type conform to the spec.
func (spec *RPCSpec) AssertOutputType(output interface{}) error {
	if !spec.Validated() {
		return fmt.Errorf("RPCSpec has not validated yet")
	}
	if outputType := reflect.TypeOf(output); outputType != spec.outputType {
		return fmt.Errorf("%s output expect %s, but got %s", spec.String(), spec.outputType.String(), outputType.String())
	}
	return nil
}

func (spec *RPCSpec) String() string {
	if !spec.Validated() {
		return fmt.Sprintf("RPCSpec(%s::%s)", spec.SvcName, spec.MethodName)
	}
	return fmt.Sprintf("RPCSpec(%s::%s %s=>%s)", spec.SvcName, spec.MethodName, spec.inputType.String(), spec.outputType.String())
}
