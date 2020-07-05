package natsrpc

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/huangjunwen/nproto/v2/enc"
	"github.com/huangjunwen/nproto/v2/enc/jsonenc"
	"github.com/huangjunwen/nproto/v2/enc/pbenc"
)

var (
	// ErrNCMaxReconnect is returned if nc has MaxReconnects >= 0.
	ErrNCMaxReconnect = errors.New("natsrpc.Server nats.Conn should have MaxReconnects < 0")

	// ErrServerClosed is returned when the server is closed
	ErrServerClosed = errors.New("natsrpc.Server closed")
)

var (
	// DefaultSubjectPrefix is the default value of ServerOptSubjectPrefix/ClientOptSubjectPrefix.
	DefaultSubjectPrefix = "natsrpc"

	// DefaultGroup is the default value of ServerOptGroup.
	DefaultGroup = "def"

	// DefaultServerEncoders is the default value of ServerOptEncoders.
	DefaultServerEncoders = []enc.Encoder{
		pbenc.Default,
		jsonenc.Default,
	}

	// DefaultClientEncoder is the default value of ClientOptEncoder.
	DefaultClientEncoder enc.Encoder = pbenc.Default

	// DefaultClientTimeout is the default value of ClientOptTimeout.
	DefaultClientTimeout time.Duration = 5 * time.Second
)

func subjectFormat(subjectPrefix, svcName, methodName string) (subject string) {
	return fmt.Sprintf("%s.%s.%s", subjectPrefix, svcName, methodName)
}

func subjectParser(subjectPrefix, svcName string) func(subject string) (ok bool, methodName string) {
	prefix := fmt.Sprintf("%s.%s.", subjectPrefix, svcName)
	return func(subject string) (bool, string) {
		methodName := strings.TrimPrefix(subject, prefix)
		if methodName == subject {
			return false, ""
		}
		return true, methodName
	}
}
