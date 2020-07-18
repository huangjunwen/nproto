// Package rpc contains high level types/interfaces for rpc implementations.
package rpc

import (
	"context"
	"fmt"
	"reflect"
	"regexp"

	"github.com/huangjunwen/nproto/v2/enc"
)

var (
	// SvcNameRegexp is service name's format.
	SvcNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)

	// MethodNameRegexp is method name's format.
	MethodNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
)

// RPCSpec is the contract between rpc server and client.
type RPCSpec interface {
	// SvcName returns the name of service.
	SvcName() string

	// MethodName returns the name of method.
	MethodName() string

	// NewInput generates a new input parameter. Must be a pointer.
	NewInput() interface{}

	// NewOutput generates a new output parameter. Must be a pointer.
	NewOutput() interface{}

	// InputType returns input's type.
	InputType() reflect.Type

	// OutputType returns output's type.
	OutputType() reflect.Type
}

type rpcSpec struct {
	svcName    string
	methodName string
	newInput   func() interface{}
	newOutput  func() interface{}
	inputType  reflect.Type
	outputType reflect.Type
}

type rawDataRPCSpec struct {
	svcName    string
	methodName string
}

// RPCHandler do the real job. RPCHandler can be client side or server side.
// It must be able to transfer normal error and RPCError from server side
// to client side.
type RPCHandler func(context.Context, interface{}) (interface{}, error)

// RPCMiddleware wraps a RPCHandler into another one.
type RPCMiddleware func(spec RPCSpec, handler RPCHandler) RPCHandler

// MustRPCSpec is must-version of NewRPCSpec.
func MustRPCSpec(svcName, methodName string, newInput, newOutput func() interface{}) RPCSpec {
	spec, err := NewRPCSpec(svcName, methodName, newInput, newOutput)
	if err != nil {
		panic(err)
	}
	return spec
}

// NewRPCSpec validates and creates a new RPCSpec.
func NewRPCSpec(svcName, methodName string, newInput, newOutput func() interface{}) (RPCSpec, error) {
	if !SvcNameRegexp.MatchString(svcName) {
		return nil, fmt.Errorf("SvcName format invalid")
	}
	if !MethodNameRegexp.MatchString(methodName) {
		return nil, fmt.Errorf("MethodName format invalid")
	}

	if newInput == nil {
		return nil, fmt.Errorf("NewInput is empty")
	}
	input := newInput()
	if input == nil {
		return nil, fmt.Errorf("NewInput() returns nil")
	}
	inputType := reflect.TypeOf(input)
	if inputType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("NewInput() returns %s which is not a pointer", inputType.String())
	}

	if newOutput == nil {
		return nil, fmt.Errorf("NewOutput is empty")
	}
	output := newOutput()
	if output == nil {
		return nil, fmt.Errorf("NewOutput() returns nil")
	}
	outputType := reflect.TypeOf(output)
	if outputType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("NewOutput() returns %s which is not a pointer", outputType.String())
	}

	return &rpcSpec{
		svcName:    svcName,
		methodName: methodName,
		newInput:   newInput,
		newOutput:  newOutput,
		inputType:  inputType,
		outputType: outputType,
	}, nil
}

func (spec *rpcSpec) SvcName() string {
	return spec.svcName
}

func (spec *rpcSpec) MethodName() string {
	return spec.methodName
}

func (spec *rpcSpec) NewInput() interface{} {
	return spec.newInput()
}

func (spec *rpcSpec) NewOutput() interface{} {
	return spec.newOutput()
}

func (spec *rpcSpec) InputType() reflect.Type {
	return spec.inputType
}

func (spec *rpcSpec) OutputType() reflect.Type {
	return spec.outputType
}

func (spec *rpcSpec) String() string {
	return fmt.Sprintf("RPCSpec(%s::%s %s=>%s)", spec.svcName, spec.methodName, spec.inputType.String(), spec.outputType.String())
}

// MustRawDataRPCSpec is must-version of NewRawDataSpec.
func MustRawDataRPCSpec(svcName, methodName string) RPCSpec {
	spec, err := NewRawDataRPCSpec(svcName, methodName)
	if err != nil {
		panic(err)
	}
	return spec
}

// NewRawDataRPCSpec validates and creates a RPCSpec which use *enc.RawData as input/output.
func NewRawDataRPCSpec(svcName, methodName string) (RPCSpec, error) {
	if !SvcNameRegexp.MatchString(svcName) {
		return nil, fmt.Errorf("SvcName format invalid")
	}
	if !MethodNameRegexp.MatchString(methodName) {
		return nil, fmt.Errorf("MethodName format invalid")
	}
	return &rawDataRPCSpec{
		svcName:    svcName,
		methodName: methodName,
	}, nil
}

func (spec *rawDataRPCSpec) SvcName() string {
	return spec.svcName
}

func (spec *rawDataRPCSpec) MethodName() string {
	return spec.methodName
}

func (spec *rawDataRPCSpec) NewInput() interface{} {
	return &enc.RawData{}
}

func (spec *rawDataRPCSpec) NewOutput() interface{} {
	return &enc.RawData{}
}

var (
	rawDataType = reflect.TypeOf((*enc.RawData)(nil))
)

func (spec *rawDataRPCSpec) InputType() reflect.Type {
	return rawDataType
}

func (spec *rawDataRPCSpec) OutputType() reflect.Type {
	return rawDataType
}

func (spec *rawDataRPCSpec) String() string {
	return fmt.Sprintf("RawDataRPCSpec(%s::%s)", spec.svcName, spec.methodName)
}

// AssertInputType makes sure input's type conform to the spec:
// reflect.TypeOf(input) == spec.InputType()
func AssertInputType(spec RPCSpec, input interface{}) error {
	if inputType := reflect.TypeOf(input); inputType != spec.InputType() {
		return fmt.Errorf("%s got unexpected input type %s", spec, inputType.String())
	}
	return nil
}

// AssertOutputType makes sure output's type conform to the spec:
// reflect.TypeOf(output) == spec.OutputType()
func AssertOutputType(spec RPCSpec, output interface{}) error {
	if outputType := reflect.TypeOf(output); outputType != spec.OutputType() {
		return fmt.Errorf("%s got unexpected output type %s", spec, outputType.String())
	}
	return nil
}
