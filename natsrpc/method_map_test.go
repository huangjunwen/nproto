package natsrpc

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"

	. "github.com/huangjunwen/nproto/v2/rpc"
)

func TestMethodMap(t *testing.T) {
	assert := assert.New(t)

	svcName := "svc"
	methodName := "method"
	testCases := []*struct {
		// Params.
		Spec         *RPCSpec
		Handler      RPCHandler
		EncoderNames map[string]struct{}

		// Store Get() result here to check later.
		GetResult map[string]*methodInfo

		// Checks.
		ExpectExists        bool
		ExpectLen           int
		ExpectHandlerResult interface{}
	}{
		/* NOTE: Test cases order is important. */
		// Regist.
		{
			Spec:                &RPCSpec{SvcName: svcName, MethodName: methodName},
			Handler:             func(_ context.Context, input interface{}) (interface{}, error) { return 1, nil },
			EncoderNames:        map[string]struct{}{"json": struct{}{}},
			ExpectExists:        true,
			ExpectHandlerResult: 1,
		},
		// Overwrite.
		{
			Spec:                &RPCSpec{SvcName: svcName, MethodName: methodName},
			Handler:             func(_ context.Context, input interface{}) (interface{}, error) { return 2, nil },
			EncoderNames:        map[string]struct{}{"pb": struct{}{}},
			ExpectExists:        true,
			ExpectHandlerResult: 2,
		},
		// Deregist.
		{
			Spec:                &RPCSpec{SvcName: svcName, MethodName: methodName},
			Handler:             (func(_ context.Context, input interface{}) (interface{}, error))(nil),
			EncoderNames:        map[string]struct{}{"pb": struct{}{}},
			ExpectExists:        false,
			ExpectHandlerResult: 0,
		},
	}

	mm := newMethodMap()
	assert.Len(mm.Get(), 0)

	// Test Regist/Deregist.
	for i, testCase := range testCases {

		mm.RegistHandler(testCase.Spec, testCase.Handler, testCase.EncoderNames)
		m := mm.Get()
		testCase.GetResult = m

		info := m[methodName]

		if !testCase.ExpectExists {
			assert.Nil(info)
			continue
		}

		assert.NotNil(info)
		assert.Equal(testCase.Spec, info.Spec)
		output, _ := info.Handler(context.Background(), nil)
		assert.Equal(testCase.ExpectHandlerResult, output, "test case %d", i)
		assert.Equal(testCase.EncoderNames, info.EncoderNames)

	}

	// Ensure Get() reuslts are always unchanged after returned.
	for i, testCase := range testCases {

		m := testCase.GetResult
		spew.Dump(m)

		info := m[methodName]

		if !testCase.ExpectExists {
			assert.Nil(info)
			continue
		}

		assert.NotNil(info)
		assert.Equal(testCase.Spec, info.Spec)
		output, _ := info.Handler(context.Background(), nil)
		assert.Equal(testCase.ExpectHandlerResult, output, "test case %d", i)
		assert.Equal(testCase.EncoderNames, info.EncoderNames)

	}

}
