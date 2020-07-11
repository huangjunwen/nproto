package stanmsg

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/taskrunner"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/rs/xid"
	"google.golang.org/protobuf/proto"

	. "github.com/huangjunwen/nproto/v2/enc"
	npmd "github.com/huangjunwen/nproto/v2/md"
	. "github.com/huangjunwen/nproto/v2/msg"
	nppb "github.com/huangjunwen/nproto/v2/pb"
)

type DurConn struct {
	// Immutable fields.
	nc            *nats.Conn
	clusterID     string
	ctx           context.Context
	logger        logr.Logger
	subjectPrefix string
	reconnectWait time.Duration
	stanOptions   []stan.Option
	runner        taskrunner.TaskRunner // runner for handlers.
	subEncoders   map[string]Encoder    // default subscriber encoders
	subRetryWait  time.Duration

	connectMu sync.Mutex // at most on connect can be run at any time

	mu sync.Mutex
	// Mutable fields.
	closed    bool
	subs      map[[2]string]*subscription // (subject, queue) -> subscriptioin
	sc        stan.Conn                   // nil if DurConn has not connected or is reconnecting
	scStaleCh chan struct{}               // pair with sc, it will closed when sc is stale
}

type subscription struct {
	spec        *MsgSpec
	queue       string
	handler     MsgHandler
	encoders    map[string]Encoder // supported encoders for this subscription
	stanOptions []stan.SubscriptionOption
}

func (dc *DurConn) connect(wait bool) {
	if wait {
		time.Sleep(dc.reconnectWait)
	}

	dc.connectMu.Lock()
	defer dc.connectMu.Unlock()

	// Reset connection: release old connection
	{
		dc.mu.Lock()
		if dc.closed {
			dc.mu.Unlock()
			dc.logger.Info("closed when reseting connection")
			return
		}
		sc := dc.sc
		scStaleCh := dc.scStaleCh
		dc.sc = nil
		dc.scStaleCh = nil
		dc.mu.Unlock()

		if sc != nil {
			sc.Close()
			close(scStaleCh)
		}
	}

	// Connect
	var sc stan.Conn
	{
		opts := []stan.Option{}
		opts = append(opts, dc.stanOptions...)
		opts = append(opts, stan.NatsConn(dc.nc))
		// NOTE: ConnectionLostHandler is used to be notified if the Streaming connection
		// is closed due to unexpected errors.
		// The callback will not be invoked on normal Conn.Close().
		opts = append(opts, stan.SetConnectionLostHandler(func(sc stan.Conn, err error) {
			dc.logger.Error(err, "connection lost")
			// reconnect after a while
			go dc.connect(true)
		}))

		// NOTE: Use a UUID-like id as client id sine we only use durable queue subscription.
		// See: https://groups.google.com/d/msg/natsio/SkWAdSU1AgU/tCX9f3ONBQAJ
		var err error
		sc, err = stan.Connect(dc.clusterID, xid.New().String(), opts...)
		if err != nil {
			// reconnect immediately.
			dc.logger.Error(err, "connect failed")
			go dc.connect(true)
			return
		}
	}

	// Update new connection
	var subs []*subscription
	scStaleCh := make(chan struct{})
	{
		dc.mu.Lock()
		if dc.closed {
			dc.mu.Unlock()
			sc.Close()
			close(scStaleCh)
			dc.logger.Info("closed when updating connection")
			return
		}
		dc.sc = sc
		dc.scStaleCh = scStaleCh
		for _, sub := range dc.subs {
			subs = append(subs, sub)
		}
		dc.mu.Unlock()
	}

	// Re-subscribe
	go dc.subscribeAll(subs, sc, scStaleCh)

}

func (dc *DurConn) subscribe(sub *subscription) error {
	key := [2]string{sub.spec.SubjectName, sub.queue}
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.closed {
		return ErrDurConnClosed
	}

	if _, ok := dc.subs[key]; ok {
		return ErrDupSubscription
	}

	// subscribe if sc is not nil
	if dc.sc != nil {
		if err := dc.subscribeOne(sub, dc.sc); err != nil {
			return err
		}
	}

	dc.subs[key] = sub
	return nil
}

func (dc *DurConn) subscribeAll(subs []*subscription, sc stan.Conn, scStaleCh chan struct{}) {

	success := make([]bool, len(subs))
	for {
		n := 0
		for i, sub := range subs {
			if success[i] {
				// Already success.
				n++
				continue
			}
			if err := dc.subscribeOne(sub, sc); err != nil {
				dc.logger.Error(err, "subscribe error", "subject", sub.spec.SubjectName, "queue", sub.queue)
				continue
			}
			dc.logger.Info("subscribe successfully", "subject", sub.spec.SubjectName, "queue", sub.queue)
			success[i] = true
			n++

			select {
			case <-scStaleCh:
				dc.logger.Info("subscribe stale")
				return
			default:
			}
		}

		if n >= len(subs) {
			// All success.
			return
		}

		select {
		case <-scStaleCh:
			dc.logger.Info("subscribe stale during retry wait")
			return

		case <-time.After(dc.subRetryWait):
		}

	}

}

func (dc *DurConn) subscribeOne(sub *subscription, sc stan.Conn) error {

	fullSubject := fmt.Sprintf("%s.%s", dc.subjectPrefix, sub.spec.SubjectName)
	opts := []stan.SubscriptionOption{}
	opts = append(opts, sub.stanOptions...)
	opts = append(opts, stan.SetManualAckMode())     // Use manual ack mode.
	opts = append(opts, stan.DurableName(sub.queue)) // Queue as durable name.
	_, err := sc.QueueSubscribe(fullSubject, sub.queue, dc.msgHandler(sub), opts...)
	return err

}

func (dc *DurConn) msgHandler(sub *subscription) stan.MsgHandler {

	logger := dc.logger.WithValues("subject", sub.spec.SubjectName, "queue", sub.queue)

	return func(m *stan.Msg) {

		if err := dc.runner.Submit(func() {
			r := &nppb.RawDataWithMD{}
			if err := proto.Unmarshal(m.Data, r); err != nil {
				logger.Error(err, "unmarshal msg error", "data", m.Data)
				return
			}

			encoderName := r.Data.EncoderName
			encoder, ok := sub.encoders[encoderName]
			if !ok {
				logger.Error(nil, "encoder not supported", "encoder", encoderName)
				return
			}

			msg := sub.spec.NewMsg()
			if err := encoder.DecodeData(bytes.NewReader(r.Data.Bytes), msg); err != nil {
				logger.Error(err, "decode msg error")
				return
			}

			ctx := dc.ctx
			if len(r.MetaData.KeyValues) != 0 {
				ctx = npmd.NewIncomingContextWithMD(ctx, r.MetaData.To())
			}

			if err := sub.handler(ctx, msg); err != nil {
				// NOTE: do not print handle's error log. Let the handler do it itself.
				return
			}

			// Ack if no error.
			m.Ack()
			return

		}); err != nil {

			logger.Error(err, "submit task error")

		}

	}

}

func (dc *DurConn) Close() {
	dc.mu.Lock()
	if dc.closed {
		dc.mu.Unlock()
		return
	}
	sc := dc.sc
	scStaleCh := dc.scStaleCh
	dc.sc = nil
	dc.scStaleCh = nil
	dc.closed = true
	dc.mu.Unlock()

	if sc != nil {
		sc.Close()
		close(scStaleCh)
	}

	dc.runner.Close()

}
