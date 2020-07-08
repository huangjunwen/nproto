package natsrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/taskrunner"

	"github.com/huangjunwen/nproto/v2/enc"
)

func ServerOptLogger(logger logr.Logger) ServerOption {
	return func(server *Server) error {
		server.logger = logger
		return nil
	}
}

// ServerOptRunner sets runner for handlers. Note that the server
// will call runner.Close() when closed.
func ServerOptRunner(runner taskrunner.TaskRunner) ServerOption {
	return func(server *Server) error {
		// Close it before replacing.
		server.runner.Close()
		server.runner = runner
		return nil
	}
}

func ServerOptSubjectPrefix(subjectPrefix string) ServerOption {
	return func(server *Server) error {
		server.subjectPrefix = subjectPrefix
		return nil
	}
}

func ServerOptGroup(group string) ServerOption {
	return func(server *Server) error {
		server.group = group
		return nil
	}
}

func ServerOptEncoders(encoders ...enc.Encoder) ServerOption {
	return func(server *Server) error {
		if len(encoders) == 0 {
			return fmt.Errorf("natsrpc.ServerOptEncoders no encoder?")
		}
		server.encoders = map[string]enc.Encoder{}
		for _, encoder := range encoders {
			server.encoders[encoder.Name()] = encoder
		}
		return nil
	}
}

func ServerOptContext(ctx context.Context) ServerOption {
	return func(server *Server) error {
		server.ctx = ctx
		return nil
	}
}

func ClientOptSubjectPrefix(subjectPrefix string) ClientOption {
	return func(client *Client) error {
		client.subjectPrefix = subjectPrefix
		return nil
	}
}

func ClientOptEncoder(encoder enc.Encoder) ClientOption {
	return func(client *Client) error {
		client.encoder = encoder
		return nil
	}
}

func ClientOptTimeout(timeout time.Duration) ClientOption {
	return func(client *Client) error {
		client.timeout = timeout
		return nil
	}
}
