package natsrpc

import (
	"context"
	"log"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"

	. "github.com/huangjunwen/nproto/v2/enc"
	"github.com/huangjunwen/nproto/v2/enc/jsonenc"
	"github.com/huangjunwen/nproto/v2/enc/pbenc"
	. "github.com/huangjunwen/nproto/v2/rpc"
)

func TestMethodMap(t *testing.T) {
	log.Printf("\n")
	log.Printf(">>> TestMethodMap.\n")
	assert := assert.New(t)

	svcName := "svc"
	methodName := "method"
	testCases := []*struct {
		// Params.
		Spec     *RPCSpec
		Handler  RPCHandler
		Encoders map[string]Encoder

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
			Encoders:            map[string]Encoder{jsonenc.Default.EncoderName(): jsonenc.Default},
			ExpectExists:        true,
			ExpectHandlerResult: 1,
		},
		// Overwrite.
		{
			Spec:                &RPCSpec{SvcName: svcName, MethodName: methodName},
			Handler:             func(_ context.Context, input interface{}) (interface{}, error) { return 2, nil },
			Encoders:            map[string]Encoder{pbenc.Default.EncoderName(): pbenc.Default},
			ExpectExists:        true,
			ExpectHandlerResult: 2,
		},
		// Deregist.
		{
			Spec:                &RPCSpec{SvcName: svcName, MethodName: methodName},
			Handler:             (func(_ context.Context, input interface{}) (interface{}, error))(nil),
			Encoders:            map[string]Encoder{pbenc.Default.EncoderName(): pbenc.Default},
			ExpectExists:        false,
			ExpectHandlerResult: 0,
		},
	}

	mm := newMethodMap()
	assert.Len(mm.Get(), 0)

	// Test Regist/Deregist.
	for i, testCase := range testCases {

		mm.RegistHandler(testCase.Spec, testCase.Handler, testCase.Encoders)
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
		assert.Equal(testCase.Encoders, info.Encoders)

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
		assert.Equal(testCase.Encoders, info.Encoders)

	}

}
