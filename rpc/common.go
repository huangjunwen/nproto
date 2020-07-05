package rpc

import (
	"context"
	"fmt"
	"reflect"
)

// RPCSpec is the contract between rpc server and client.
// It should be readonly. Don't change its content once it is filled.
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

// AssertInputType makes sure input's type conform to the spec.
// It will panic if NewInput() returns nil or non-pointer, or input's type is different.
func (spec *RPCSpec) AssertInputType(input interface{}) {
	if spec.inputType == nil {
		newInput := spec.NewInput()
		if newInput == nil {
			panic(fmt.Errorf("%s NewInput() returns nil", spec.String()))
		}
		inputType := reflect.TypeOf(newInput)
		if inputType.Kind() != reflect.Ptr {
			panic(fmt.Errorf("%s NewInput() returns non-pointer", spec.String()))
		}
		spec.inputType = inputType
	}
	if inputType := reflect.TypeOf(input); inputType != spec.inputType {
		panic(fmt.Errorf("%s input expect %s, but got %s", spec.String(), spec.inputType.String(), inputType.String()))
	}
}

// AssertOutputType makes sure output's type conform to the spec.
// It will panic if NewOutput() returns nil or non-pointer, or output's type is different.
func (spec *RPCSpec) AssertOutputType(output interface{}) {
	if spec.outputType == nil {
		newOutput := spec.NewOutput()
		if newOutput == nil {
			panic(fmt.Errorf("%s NewOutput() returns nil", spec.String()))
		}
		outputType := reflect.TypeOf(newOutput)
		if outputType.Kind() != reflect.Ptr {
			panic(fmt.Errorf("%s NewOutput() returns non-pointer", spec.String()))
		}
		spec.outputType = outputType
	}
	if outputType := reflect.TypeOf(output); outputType != spec.outputType {
		panic(fmt.Errorf("%s output expect %s, but got %s", spec.String(), spec.outputType.String(), outputType.String()))
	}
}

func (spec *RPCSpec) String() string {
	return fmt.Sprintf("RPCSpec(%s::%s)", spec.SvcName, spec.MethodName)
}
