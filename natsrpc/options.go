package natsrpc

import (
	"context"
	"fmt"
	"regexp"
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

var (
	subjectPrefixRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

// SCOptLogger sets logger for ServerConn.
func SCOptLogger(logger logr.Logger) ServerConnOption {
	return func(sc *ServerConn) error {
		if logger == nil {
			logger = logr.Nop
		}
		sc.logger = logger.WithValues("component", "nproto.natsrpc.ServerConn")
		return nil
	}
}

// SCOptRunner sets runner for handlers. Note that the runner will be closed in ServerConn.Close().
func SCOptRunner(runner taskrunner.TaskRunner) ServerConnOption {
	return func(sc *ServerConn) error {
		if runner == nil {
			return fmt.Errorf("SCOptRunner got nil taskrunner.TaskRunner")
		}
		// Close it before replacing.
		sc.runner.Close()
		sc.runner = runner
		return nil
	}
}

// SCOptSubjectPrefix sets subject prefix in nats namespace.
func SCOptSubjectPrefix(subjectPrefix string) ServerConnOption {
	return func(sc *ServerConn) error {
		if !subjectPrefixRegexp.MatchString(subjectPrefix) {
			return fmt.Errorf("SCOptSubjectPrefix got invalid subject prefix %q", subjectPrefix)
		}
		sc.subjectPrefix = subjectPrefix
		return nil
	}
}

// SCOptGroup sets nats subscription group name.
func SCOptGroup(group string) ServerConnOption {
	return func(sc *ServerConn) error {
		if group == "" {
			return fmt.Errorf("SCOptGroup got empty group")
		}
		sc.group = group
		return nil
	}
}

// SCOptContext sets base context used in handlers.
func SCOptContext(ctx context.Context) ServerConnOption {
	return func(sc *ServerConn) error {
		if ctx == nil {
			return fmt.Errorf("SCOptContext got nil context.Context")
		}
		sc.ctx = ctx
		return nil
	}
}

// CCOptSubjectPrefix sets subject prefix in nats namespace.
func CCOptSubjectPrefix(subjectPrefix string) ClientConnOption {
	return func(cc *ClientConn) error {
		if !subjectPrefixRegexp.MatchString(subjectPrefix) {
			return fmt.Errorf("CCOptSubjectPrefix got invalid subject prefix %q", subjectPrefix)
		}
		cc.subjectPrefix = subjectPrefix
		return nil
	}
}

// CCOptTimeout sets the default timeout when invoking rpc if there is no timeout set in context.
func CCOptTimeout(t time.Duration) ClientConnOption {
	return func(cc *ClientConn) error {
		if t <= 0 {
			return fmt.Errorf("CCOptTimeout got non-positive duration %s", t.String())
		}
		cc.timeout = t
		return nil
	}
}

func init() {
	if !subjectPrefixRegexp.MatchString(DefaultSubjectPrefix) {
		panic(fmt.Errorf("DefaultSubjectPrefix %q invalid", DefaultSubjectPrefix))
	}
}
