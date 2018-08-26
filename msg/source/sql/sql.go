package sqlsrc

import (
	"context"
	"database/sql"

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
	_ libmsg.MsgSource = (*SQLMsgSource)(nil)
	_ libmsg.Msg       = (*sqlMsg)(nil)
	_ Queryer          = (*sql.DB)(nil)
	_ Queryer          = (*sql.Conn)(nil)
	_ Queryer          = (*sql.Tx)(nil)
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

// Store stores a message to be delivered. Usually this method is called inside a tx
// so that the message is committed together with other data.
func (src *SQLMsgSource) Store(ctx context.Context, q Queryer, subject string, data []byte) error {
	_, err := q.ExecContext(ctx, src.dialect.InsertStmt(), subject, data)
	return err
}

// Fetch implements libmsg.MsgSource interface.
func (src *SQLMsgSource) Fetch() <-chan libmsg.Msg {

	logger := src.logger.With().Str("fn", "Fetch").Logger()

	ret := make(chan libmsg.Msg)
	rows, err := src.db.Query(src.dialect.SelectStmt())
	if err != nil {
		logger.Error().Err(err).Msg("")
		close(ret)
		return ret
	}

	go func() {
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
	}()

	return ret
}

// DeliverResult implements libmsg.MsgSource interface.
func (src *SQLMsgSource) DeliverResult(msgs []libmsg.Msg, delivered []bool) {

	logger := src.logger.With().Str("fn", "DeliverResult").Logger()

	ids := []interface{}{}
	for i, msg := range msgs {
		if !delivered[i] {
			continue
		}
		ids = append(ids, msg.(*sqlMsg).id)
	}

	if len(ids) == 0 {
		return
	}

	_, err := src.db.Exec(src.dialect.DeleteStmt(len(ids)), ids...)
	if err != nil {
		logger.Error().Err(err).Msg("")
	}

}

// Subject implements libmsg.Msg interface.
func (m *sqlMsg) Subject() string {
	return m.subject
}

// Data implements libmsg.Msg interface.
func (m *sqlMsg) Data() []byte {
	return m.data
}

func SQLMsgSourceOptLogger(logger *zerolog.Logger) SQLMsgSourceOption {
	return func(src *SQLMsgSource) {
		if logger == nil {
			nop := zerolog.Nop()
			logger = &nop
		}
		src.logger = logger.With().Str("comp", "nproto.libmsg.source.sql.SQLMsgSource").Logger()
	}
}
