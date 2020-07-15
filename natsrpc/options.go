package natsrpc

import (
	"context"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/taskrunner"
)

var (
	// DefaultSubjectPrefix is the default value of SCOptSubjectPrefix/CCOptSubjectPrefix.
	DefaultSubjectPrefix = "natsrpc"

	// DefaultGroup is the default value of SCOptGroup.
	DefaultGroup = "def"

	// DefaultClientTimeout is the default value of CCOptTimeout.
	DefaultClientTimeout time.Duration = 5 * time.Second
)

func SCOptLogger(logger logr.Logger) ServerConnOption {
	return func(sc *ServerConn) error {
		sc.logger = logger.WithValues("component", "nproto.natsrpc.ServerConn")
		return nil
	}
}

// SCOptRunner sets runner for handlers. Note that the runner will be closed in ServerConn.Close().
func SCOptRunner(runner taskrunner.TaskRunner) ServerConnOption {
	return func(sc *ServerConn) error {
		// Close it before replacing.
		sc.runner.Close()
		sc.runner = runner
		return nil
	}
}

func SCOptSubjectPrefix(subjectPrefix string) ServerConnOption {
	return func(sc *ServerConn) error {
		sc.subjectPrefix = subjectPrefix
		return nil
	}
}

func SCOptGroup(group string) ServerConnOption {
	return func(sc *ServerConn) error {
		sc.group = group
		return nil
	}
}

func SCOptContext(ctx context.Context) ServerConnOption {
	return func(sc *ServerConn) error {
		sc.ctx = ctx
		return nil
	}
}

func CCOptSubjectPrefix(subjectPrefix string) ClientConnOption {
	return func(cc *ClientConn) error {
		cc.subjectPrefix = subjectPrefix
		return nil
	}
}

func CCOptTimeout(timeout time.Duration) ClientConnOption {
	return func(cc *ClientConn) error {
		cc.timeout = timeout
		return nil
	}
}
