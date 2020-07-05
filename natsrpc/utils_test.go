package natsrpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubjectFormatParse(t *testing.T) {
	assert := assert.New(t)

	svcName := "test"
	methodName := "ping"
	subject := subjectFormat(DefaultSubjectPrefix, svcName, methodName)
	fmt.Println(subject)
	{
		ok, methodName2 := subjectParser(DefaultSubjectPrefix, svcName)(subject)
		assert.True(ok)
		assert.Equal(methodName, methodName2)
	}

	{
		ok, _ := subjectParser(DefaultSubjectPrefix, svcName)("natsrpc.prod.ping")
		assert.False(ok)
	}
}
