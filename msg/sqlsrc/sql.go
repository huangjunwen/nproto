package sqlsrc

import (
	"context"
	"database/sql"

	"github.com/golang/protobuf/proto"
	"github.com/rs/zerolog"

	"github.com/huangjunwen/nproto/msg"
)

// SQLMsgSource use RDBMS to store messages.
type SQLMsgSource struct {
	db        *sql.DB
	dialect   sqlMsgSourceDialect
	tableName string

	// Options.
	logger zerolog.Logger
}

// Queryer abstracts sql.DB/sql.Conn/sql.Tx .
type Queryer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type sqlMsgSourceDialect interface {
	CreateStmt() string
	InsertStmt() string
	SelectStmt() string
	DeleteStmt(n int) string
}

type sqlMsg struct {
	id      int
	subject string
	data    []byte
}

// SQLMsgSourceOption is options in creating SQLMsgSource.
type SQLMsgSourceOption func(*SQLMsgSource)

var (
	_ npmsg.MsgSource = (*SQLMsgSource)(nil)
	_ npmsg.Msg       = (*sqlMsg)(nil)
	_ Queryer         = (*sql.DB)(nil)
	_ Queryer         = (*sql.Conn)(nil)
	_ Queryer         = (*sql.Tx)(nil)
)

func newSQLMsgSource(dialectFactory func(string) sqlMsgSourceDialect, db *sql.DB, tableName string, opts ...SQLMsgSourceOption) (*SQLMsgSource, error) {

	ret := &SQLMsgSource{
		db:        db,
		dialect:   dialectFactory(tableName),
		tableName: tableName,
		logger:    zerolog.Nop(),
	}
	for _, opt := range opts {
		opt(ret)
	}

	_, err := db.Exec(ret.dialect.CreateStmt())
	if err != nil {
		return nil, err
	}

	return ret, nil

}

// Publish stores a message to be delivered. Usually this method is called inside a tx
// so that the message is committed together with other data.
func (src *SQLMsgSource) Publish(ctx context.Context, q Queryer, subject string, data []byte) error {
	_, err := q.ExecContext(ctx, src.dialect.InsertStmt(), subject, data)
	return err
}

// PublishProto stores a protobuf message to be delivered.
func (src *SQLMsgSource) PublishProto(ctx context.Context, q Queryer, subject string, m proto.Message) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	return src.Publish(ctx, q, subject, data)
}

// Fetch implements npmsg.MsgSource interface.
func (src *SQLMsgSource) Fetch() <-chan npmsg.Msg {

	logger := src.logger.With().Str("fn", "Fetch").Logger()

	ret := make(chan npmsg.Msg)
	rows, err := src.db.Query(src.dialect.SelectStmt())
	if err != nil {
		logger.Error().Err(err).Msg("")
		close(ret)
		return ret
	}

	cfh.Go("SQLMsgSource.Fetch", func() {
		defer close(ret)
		defer rows.Close()
		for rows.Next() {
			m := &sqlMsg{}
			if err := rows.Scan(&m.id, &m.subject, &m.data); err != nil {
				logger.Error().Err(err).Msg("")
				break
			}
			ret <- m
		}
	})

	return ret
}

// ProcessPublishMsgsResult implements npmsg.MsgSource interface.
func (src *SQLMsgSource) ProcessPublishMsgsResult(msgs []npmsg.Msg, errors []error) {

	ids := []interface{}{}
	for i, msg := range msgs {
		if errors[i] != nil {
			continue
		}
		ids = append(ids, msg.(*sqlMsg).id)
	}

	if len(ids) == 0 {
		return
	}

	_, err := src.db.Exec(src.dialect.DeleteStmt(len(ids)), ids...)
	if err != nil {
		src.logger.Error().Str("fn", "ProcessPublishBatchResult").Err(err).Msg("")
	}

}

// Subject implements npmsg.Msg interface.
func (m *sqlMsg) Subject() string {
	return m.subject
}

// Data implements npmsg.Msg interface.
func (m *sqlMsg) Data() []byte {
	return m.data
}

func SQLMsgSourceOptLogger(logger *zerolog.Logger) SQLMsgSourceOption {
	return func(src *SQLMsgSource) {
		if logger == nil {
			nop := zerolog.Nop()
			logger = &nop
		}
		src.logger = logger.With().Str("comp", "nproto.npmsg.sqlsrc.SQLMsgSource").Logger()
	}
}
