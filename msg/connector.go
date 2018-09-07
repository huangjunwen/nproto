package npmsg

import (
	"time"

	"github.com/rs/zerolog"
)

var (
	DefaultMsgConnectorBatch         = 500
	DefaultMsgConnectorFetchInterval = 30 * time.Second
)

// MsgSource is the source of messages.
type MsgSource interface {
	// Fetch returns a channel to get messages from the source.
	Fetch() <-chan Msg

	// ProcessPublishMsgsResult is called after publish a batch of messages.
	// `errors[i]` is nil if `msgs[i]` has been delivered successfully so that the
	// message can be deleted.
	ProcessPublishMsgsResult(msgs []Msg, errors []error)
}

// MsgConnector deliver messages from source to a publisher.
type MsgConnector struct {
	src       MsgSource
	publisher MsgPublisher
	stopC     chan struct{}
	kickC     chan struct{}

	// Options.
	batch         int
	fetchInterval time.Duration
	logger        zerolog.Logger
}

// MsgConnectorOption is the option in creating MsgConnector.
type MsgConnectorOption func(*MsgConnector) error

// NewMsgConnector creates a new MsgConnector.
func NewMsgConnector(src MsgSource, publisher MsgPublisher, opts ...MsgConnectorOption) (*MsgConnector, error) {
	ret := &MsgConnector{
		src:           src,
		publisher:     publisher,
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

				// Load a batch of messages each time.
				for msg := range msgC {
					msgs = append(msgs, msg)
					if len(msgs) >= connector.batch {
						break
					}
				}
				if len(msgs) == 0 {
					// No message at the moment.
					break
				}

				// Deliver.
				errors := connector.publisher.PublishMsgs(msgs)

				// Process deliver result.
				connector.src.ProcessPublishMsgsResult(msgs, errors)

				// Update statistics.
				nMsgs += len(msgs)
				for _, err := range errors {
					if err == nil {
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
		connector.logger = logger.With().Str("comp", "nproto.npmsg.MsgConnector").Logger()
		return nil
	}
}
