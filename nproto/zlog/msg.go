package zlog

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/huangjunwen/nproto/nproto"
)

// WrapMsgSubscriber adds a zerolog.Logger to MsgSubscriber's context.
func WrapMsgSubscriber(logger *zerolog.Logger) nproto.MsgMiddleware {
	return func(subject, queue string, handler nproto.MsgHandler) nproto.MsgHandler {
		return func(ctx context.Context, msgData []byte) (err error) {
			// Create a copy of the logger (including internal context slice)
			// to prevent data race when using UpdateContext.
			l := logger.With().Logger()
			ctx = l.WithContext(ctx)
			return handler(ctx, msgData)
		}
	}
}
