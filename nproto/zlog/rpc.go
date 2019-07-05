package zlog

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/rs/zerolog"

	"github.com/huangjunwen/nproto/nproto"
)

// WrapRPCServer adds a zerolog.Logger to RPCServer's context.
func WrapRPCServer(logger *zerolog.Logger) nproto.RPCMiddleware {
	return func(svcName string, method *nproto.RPCMethod, handler nproto.RPCHandler) nproto.RPCHandler {
		return func(ctx context.Context, input proto.Message) (output proto.Message, err error) {
			// Create a copy of the logger (including internal context slice)
			// to prevent data race when using UpdateContext.
			l := logger.With().Logger()
			ctx = l.WithContext(ctx)
			return handler(ctx, input)
		}
	}
}
