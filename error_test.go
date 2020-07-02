package nproto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorCode(t *testing.T) {
	assert := assert.New(t)

	assert.False(ProtocolError.Retryable())
	fmt.Printf("%s\n", &Error{Code: ProtocolError, Message: "xxx"})

	assert.False(InvalidError.Retryable())
	fmt.Printf("%s\n", &Error{Code: InvalidError})

	assert.False(NotRetryableError.Retryable())
	fmt.Printf("%s\n", &Error{Code: NotRetryableError})

	assert.True(ErrorCode(-32499).Retryable())

}