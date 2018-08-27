package libmsg

import (
	"time"

	"github.com/rs/zerolog"
)

var (
	DefaultMsgConnectorBatch         = 500
	DefaultMsgConnectorFetchInterval = 30 * time.Second
)

// Msg represents a message to be delivered.
type Msg interface {
	// Subject is the topic of the message.
	Subject() string

	// Data is the payload of the message.
	Data() []byte
}

// MsgSource is the source of messages.
type MsgSource interface {
	// Fetch returns a channel to get messages.
	Fetch() <-chan Msg

	// DeliverResult is called after delivering a batch of messages.
	// `delivered[i]` is true if `msgs[i]` has been delivered successfully.
	DeliverResult(msgs []Msg, delivered []bool)
}

// MsgSink is the sink of messages.
type MsgSink interface {
	// Deliver is used to deliver messages to the sink.
	// Set `delivered[i]` to true if `msgs[i]` is successfully delivered.
	Deliver(msgs []Msg, delivered []bool)
}

// MsgConnector deliver messages from source to sink.
type MsgConnector struct {
	src   MsgSource
	sink  MsgSink
	stopC chan struct{}
	kickC chan struct{}

	// Options.
	batch         int
	fetchInterval time.Duration
	logger        zerolog.Logger
}

// MsgConnectorOption is the option in creating MsgConnector.
type MsgConnectorOption func(*MsgConnector) error

// NewMsgConnector creates a new MsgConnector.
func NewMsgConnector(src MsgSource, sink MsgSink, opts ...MsgConnectorOption) (*MsgConnector, error) {
	ret := &MsgConnector{
		src:           src,
		sink:          sink,
		stopC:         make(chan struct{}),
		kickC:         make(chan struct{}, 1),
		batch:         DefaultMsgConnectorBatch,
		fetchInterval: DefaultMsgConnectorFetchInterval,
		logger:        zerolog.Nop(),
	}
	for _, opt := range opts {
		if err := opt(ret); err != nil {
			return nil, err
		}
	}
	ret.loop()
	return ret, nil
}

func (connector *MsgConnector) loop() {

	cfh.Go("MsgConnector.loop", func() {

		stopped := false
		for !stopped {

			// Get the fetch channel.
			msgC := connector.src.Fetch()

			// Some statistics.
			nMsgs := 0
			nSuccess := 0
			startTime := time.Now()

			for {
				msgs := []Msg{}
				delivered := []bool{}

				// Load a batch of messages each time.
				for msg := range msgC {
					msgs = append(msgs, msg)
					delivered = append(delivered, false)
					if len(msgs) >= connector.batch {
						break
					}
				}
				if len(msgs) == 0 {
					// No message at the moment.
					break
				}

				// Deliver.
				connector.sink.Deliver(msgs, delivered)

				// Process deliver result.
				connector.src.DeliverResult(msgs, delivered)

				// Update statistics.
				nMsgs += len(msgs)
				for _, r := range delivered {
					if r {
						nSuccess += 1
					}
				}

			}

			dur := time.Since(startTime)
			connector.logger.Info().Int("nmsgs", nMsgs).Int("nsuccess", nSuccess).Str("dur", dur.String()).Msg("")

			// Wait for events.
			t := time.NewTimer(connector.fetchInterval)
			select {
			case <-connector.stopC:
				stopped = true
			case <-t.C:
			case <-connector.kickC:
			}
			t.Stop()

		}

	})

}

// Kick the connector to deliver messages.
func (connector *MsgConnector) Kick() {
	// Kick the connector (non-blocking) to run.
	select {
	case connector.kickC <- struct{}{}:
	default:
	}
}

// Stop the deliver loop.
func (connector *MsgConnector) Stop() {
	close(connector.stopC)
}

// MsgConnectorOptBatch sets the maximum in flight messages.
func MsgConnectorOptBatch(batch int) MsgConnectorOption {
	return func(connector *MsgConnector) error {
		connector.batch = batch
		return nil
	}
}

// MsgConnectorOptFetchInterval sets the interval connector fetch messages from source.
func MsgConnectorOptFetchInterval(fetchInterval time.Duration) MsgConnectorOption {
	return func(connector *MsgConnector) error {
		connector.fetchInterval = fetchInterval
		return nil
	}
}

// MsgConnectorOptLogger sets the logger.
func MsgConnectorOptLogger(logger *zerolog.Logger) MsgConnectorOption {
	return func(connector *MsgConnector) error {
		if logger == nil {
			nop := zerolog.Nop()
			logger = &nop
		}
		connector.logger = logger.With().Str("comp", "nproto.libmsg.MsgConnector").Logger()
		return nil
	}
}
