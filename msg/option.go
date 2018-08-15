package msg

import (
	"time"

	"github.com/nats-io/go-nats-streaming"
	"github.com/rs/zerolog"
)

var (
	// Default interval between reconnections.
	DefaultReconnectWait = 5 * time.Second
	// Default interval between resubscriptions.
	DefaultResubscribeWait = 5 * time.Second
)

// Options is options to create DurConn.
type Options struct {
	stanOptions []stan.Option

	reconnectWait time.Duration
	logger        zerolog.Logger
}

// SubscriptionOptions is options to create subscription.
type SubscriptionOptions struct {
	stanOptions []stan.SubscriptionOption

	resubscribeWait time.Duration
}

// Option is single option of Options.
type Option func(*Options)

// SubscriptionOption is single option of SubscriptionOptions.
type SubscriptionOption func(*SubscriptionOptions)

// NewOptions returns a default Options.
func NewOptions() Options {
	return Options{
		stanOptions:   []stan.Option{},
		reconnectWait: DefaultReconnectWait,
		logger:        zerolog.Nop(),
	}
}

// OptConnectWait sets connection wait.
func OptConnectWait(t time.Duration) Option {
	return func(o *Options) {
		o.stanOptions = append(o.stanOptions, stan.ConnectWait(t))
	}
}

// OptPubAckWait sets publish ack time wait.
func OptPubAckWait(t time.Duration) Option {
	return func(o *Options) {
		o.stanOptions = append(o.stanOptions, stan.PubAckWait(t))
	}
}

// OptPings sets ping
func OptPings(interval, maxOut int) Option {
	return func(o *Options) {
		o.stanOptions = append(o.stanOptions, stan.Pings(interval, maxOut))
	}
}

// OptReconnectWait sets reconnection wait.
func OptReconnectWait(t time.Duration) Option {
	return func(o *Options) {
		o.reconnectWait = t
	}
}

// OptLogger sets logger.
func OptLogger(l *zerolog.Logger) Option {
	return func(o *Options) {
		if l == nil {
			o.logger = zerolog.Nop()
		} else {
			o.logger = *l
		}
	}
}

// NewSubscriptionOptions returns a default options.
func NewSubscriptionOptions() SubscriptionOptions {
	return SubscriptionOptions{
		stanOptions:     []stan.SubscriptionOption{},
		resubscribeWait: DefaultResubscribeWait,
	}
}

// SubsOptMaxInflight sets max message that can be sent to subscriber before ack
func SubsOptMaxInflight(m int) SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.stanOptions = append(o.stanOptions, stan.MaxInflight(m))
	}
}

// SubsOptAckWait sets server side ack wait.
func SubsOptAckWait(t time.Duration) SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.stanOptions = append(o.stanOptions, stan.AckWait(t))
	}
}

// SubsOptManualAcks sets to manual ack mode.
func SubsOptManualAcks() SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.stanOptions = append(o.stanOptions, stan.SetManualAckMode())
	}
}

// SubsOptResubscribeWait sets resubscriptions wait.
func SubsOptResubscribeWait(t time.Duration) SubscriptionOption {
	return func(o *SubscriptionOptions) {
		o.resubscribeWait = t
	}
}
