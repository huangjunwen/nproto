package nproto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorCode(t *testing.T) {
	assert := assert.New(t)

	assert.False(ParseError.Retryable())

	assert.False(InvalidError.Retryable())

	assert.False(NotRetryableError.Retryable())

	assert.True(ErrorCode(-32499).Retryable())

}
