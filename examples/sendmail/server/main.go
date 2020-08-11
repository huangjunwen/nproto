package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/huangjunwen/golibs/logr"
	"github.com/huangjunwen/golibs/logr/zerologr"
	"github.com/huangjunwen/golibs/mycanal"
	"github.com/huangjunwen/golibs/sqlh"
	"github.com/nats-io/nats.go"
	ot "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"gopkg.in/gomail.v2"

	"github.com/huangjunwen/nproto/v2/binlogmsg"
	"github.com/huangjunwen/nproto/v2/enc/rawenc"
	npmsg "github.com/huangjunwen/nproto/v2/msg"
	"github.com/huangjunwen/nproto/v2/natsrpc"
	nprpc "github.com/huangjunwen/nproto/v2/rpc"
	"github.com/huangjunwen/nproto/v2/stanmsg"
	msgtracing "github.com/huangjunwen/nproto/v2/tracing/msg"
	rpctracing "github.com/huangjunwen/nproto/v2/tracing/rpc"

	"github.com/huangjunwen/nproto/v2/examples/sendmail"
)

type Config struct {
	SMTPHost string `json:"smtpHost"`
	SMTPPort int    `json:"smtpPort"`
	User     string `json:"user"`
	Password string `json:"password"`
}

const (
	// See docker-compose.yml.
	natsHost               = "localhost"
	natsPort               = 4222
	stanCluster            = "example_cluster"
	mysqlHost              = "localhost"
	mysqlPort              = 3306
	mysqlUser              = "root"
	mysqlPassword          = "123456"
	jaegerServiceName      = "sendmail"
	jaegerReporterEndpoint = "http://localhost:14268/api/traces"
)

const (
	mysqlDBName              = "sendmail"
	mysqlMsgTableName        = "_msg"
	mysqlEmailEntryTableName = "email_entry"
)

func main() {
	var logger logr.Logger
	{
		out := zerolog.NewConsoleWriter()
		out.TimeFormat = time.RFC3339
		out.Out = os.Stderr
		lg := zerolog.New(&out).With().Timestamp().Logger()
		logger = (*zerologr.Logger)(&lg)
	}

	var err error

	wg := &sync.WaitGroup{}

	stopCtx, stopFunc := context.WithCancel(context.Background())
	defer stopFunc()
	{
		// Stop the context when receiving signals.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			defer signal.Stop(sigCh)
			<-sigCh
			stopFunc()
		}()
	}

	confName := flag.String("conf", "conf.json", "Config file name (json format)")
	flag.Parse()
	var config = &Config{}
	{
		f, err := os.Open(*confName)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		content, err := ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(content, config)
		if err != nil {
			panic(err)
		}
		logger.Info("Config read ok")
	}

	var nc *nats.Conn
	{
		opts := nats.GetDefaultOptions()
		opts.Url = fmt.Sprintf("nats://%s:%d", natsHost, natsPort)
		opts.MaxReconnect = -1 // Never give up reconnect.
		nc, err = opts.Connect()
		if err != nil {
			panic(err)
		}
		defer nc.Close()
		logger.Info("Connect to nats server ok")
	}

	var db *sql.DB
	{
		config := mysql.NewConfig()
		config.User = mysqlUser
		config.Passwd = mysqlPassword
		config.Net = "tcp"
		config.Addr = fmt.Sprintf("%s:%d", mysqlHost, mysqlPort)
		config.Params = map[string]string{
			"interpolateParams": "true",
		}
		db, err = sql.Open("mysql", config.FormatDSN())
		if err != nil {
			panic(err)
		}
		defer db.Close()
		logger.Info("Open mysql client ok")
	}

	var tracer ot.Tracer
	{
		var closer io.Closer
		config := &jaegercfg.Configuration{
			ServiceName: jaegerServiceName,
			Sampler: &jaegercfg.SamplerConfig{
				Type:  "const",
				Param: 1,
			},
			// Reporter: &jaegercfg.ReporterConfig{
			// 	CollectorEndpoint: jaegerReporterEndpoint,
			// },
		}
		tracer, closer, err = config.NewTracer()
		if err != nil {
			panic(err)
		}
		defer closer.Close()
		logger.Info("New tracer ok")
	}

	// --- Components ---

	var dc *stanmsg.DurConn
	{
		dc, err = stanmsg.NewDurConn(
			nc,
			stanCluster,
			stanmsg.DCOptLogger(logger),
			stanmsg.DCOptContext(stopCtx),
		)
		if err != nil {
			panic(err)
		}
		defer dc.Close()
		logger.Info("New stanmsg.DurConn ok")
	}

	subscriber := npmsg.NewMsgSubscriberWithMWs(
		stanmsg.NewPbJsonSubscriber(dc),
		msgtracing.WrapMsgSubscriber(tracer),
	)
	logger.Info("New subscriber ok")

	var pipe *binlogmsg.MsgPipe
	{
		downstream := npmsg.NewMsgAsyncPublisherWithMWs(
			dc.NewPublisher(rawenc.DefaultRawEncoder),
			msgtracing.WrapMsgAsyncPublisher(tracer, true),
		)

		c := mycanal.Config{
			Host:     mysqlHost,
			Port:     mysqlPort,
			User:     mysqlUser,
			Password: mysqlPassword,
		}
		masterCfg := &mycanal.FullDumpConfig{Config: c}
		slaveCfg := &mycanal.IncrDumpConfig{
			Config:   c,
			ServerId: 1023,
		}

		pipe, err = binlogmsg.NewMsgPipe(
			downstream,
			masterCfg,
			slaveCfg,
			func(schema, table string) bool {
				return schema == mysqlDBName && table == mysqlMsgTableName
			},
			binlogmsg.PipeOptLogger(logger),
		)
		if err != nil {
			panic(err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			pipe.Run(stopCtx)
		}()
		logger.Info("New binlogmsg.MsgPipe ok")
	}

	newPublisher := func(q sqlh.Queryer) npmsg.MsgPublisherFunc {
		return npmsg.NewMsgPublisherWithMWs(
			binlogmsg.NewPbJsonPublisher(q, mysqlDBName, mysqlMsgTableName),
			msgtracing.WrapMsgPublisher(tracer, false),
		)
	}
	logger.Info("New newPublisher ok")

	var sc *natsrpc.ServerConn
	{
		sc, err = natsrpc.NewServerConn(
			nc,
			natsrpc.SCOptLogger(logger),
			natsrpc.SCOptContext(stopCtx),
		)
		if err != nil {
			panic(err)
		}
		defer sc.Close()
		logger.Info("New natsrpc.ServerConn ok")
	}

	var server nprpc.RPCServer
	{
		server = nprpc.NewRPCServerWithMWs(
			natsrpc.NewPbJsonServer(sc),
			rpctracing.WrapRPCServer(tracer),
		)
		logger.Info("New server ok")
	}

	// --- Prepare ---

	if err := createDatabase(stopCtx, db); err != nil {
		panic(err)
	}
	logger.Info("createDatabase  ok")

	if err := binlogmsg.CreateMsgTable(stopCtx, db, mysqlDBName, mysqlMsgTableName); err != nil {
		panic(err)
	}
	logger.Info("CreateMsgTable ok")

	if err := createEmailEntryTable(stopCtx, db); err != nil {
		panic(err)
	}
	logger.Info("createEmailEntryTable ok")

	// --- Regist rpc service and subscribe msg handler---

	svc := &sendMailSvc{
		config:       config,
		db:           db,
		newPublisher: newPublisher,
	}

	if err := sendmail.ServeSendMailSvc(server, sendmail.SvcSpec, svc); err != nil {
		panic(err)
	}
	logger.Info("ServeSendMailSvc ok")

	if err := sendmail.SubscribeEmailEntry(subscriber, sendmail.QueueSpec, "default", svc.handleEmailEntry); err != nil {
		panic(err)
	}
	logger.Info("SubscribeEmailEntry ok")

	// --- Wait ---
	<-stopCtx.Done()
}

type sendMailSvc struct {
	config       *Config
	db           *sql.DB
	newPublisher func(sqlh.Queryer) npmsg.MsgPublisherFunc
}

var (
	_ sendmail.SendMailSvc = (*sendMailSvc)(nil)
)

func (svc *sendMailSvc) Send(ctx context.Context, email *sendmail.Email) (*sendmail.EmailEntry, error) {
	tx, err := svc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// NOTE: email's fields are not checked here for simplicity since it's only a demo.
	entry, err := insertEmailEntry(ctx, tx, email)
	if err != nil {
		return nil, err
	}

	err = sendmail.PublishEmailEntry(
		svc.newPublisher(tx),
		ctx,
		sendmail.QueueSpec,
		entry,
	)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (svc *sendMailSvc) handleEmailEntry(ctx context.Context, entry *sendmail.EmailEntry) error {
	m := gomail.NewMessage()
	m.SetHeader("From", svc.config.User)
	m.SetAddressHeader("To", entry.Email.ToAddr, entry.Email.ToName)
	m.SetHeader("Subject", entry.Email.Subject)
	m.SetBody(entry.Email.ContentType, entry.Email.Content)

	d := gomail.NewDialer(svc.config.SMTPHost, svc.config.SMTPPort, svc.config.User, svc.config.Password)
	sendErr := d.DialAndSend(m)

	tx, err := svc.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := updateEmailEntry(ctx, tx, entry.Id, sendErr); err != nil {
		return err
	}

	return tx.Commit()
}

func (svc *sendMailSvc) List(ctx context.Context, _ *empty.Empty) (*sendmail.EmailEntries, error) {
	return selectEmailEntries(ctx, svc.db)
}

func createDatabase(ctx context.Context, q sqlh.Queryer) error {
	sql := fmt.Sprintf(`
	CREATE DATABASE IF NOT EXISTS %s
	`, mysqlDBName)
	_, err := q.ExecContext(ctx, sql)
	return err
}

func createEmailEntryTable(ctx context.Context, q sqlh.Queryer) error {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s.%s (
		id BIGINT NOT NULL AUTO_INCREMENT,
		status TINYINT UNSIGNED NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT NOW(),
		ended_at DATETIME NOT NULL DEFAULT '2000-01-01 00:00:00',
		failed_reason VARCHAR(256) NOT NULL DEFAULT '',
		to_addr VARCHAR(256) NOT NULL DEFAULT '',
		to_name VARCHAR(256) NOT NULL DEFAULT '',
		subject VARCHAR(256) NOT NULL DEFAULT '',
		content_type VARCHAR(64) NOT NULL DEFAULT '',
		content TEXT DEFAULT (''),
		PRIMARY KEY (id))
	`, mysqlDBName, mysqlEmailEntryTableName)
	_, err := q.ExecContext(ctx, sql)
	return err
}

func insertEmailEntry(ctx context.Context, q sqlh.Queryer, email *sendmail.Email) (*sendmail.EmailEntry, error) {
	r, err := q.ExecContext(
		ctx,
		fmt.Sprintf("INSERT INTO %s.%s (status, to_addr, to_name, subject, content_type, content) VALUES (?, ?, ?, ?, ?, ?)", mysqlDBName, mysqlEmailEntryTableName),
		sendmail.EmailEntry_SENDING,
		email.ToAddr,
		email.ToName,
		email.Subject,
		email.ContentType,
		email.Content,
	)
	if err != nil {
		return nil, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return nil, err
	}
	return selectEmailEntry(ctx, q, id)
}

func updateEmailEntry(ctx context.Context, q sqlh.Queryer, id int64, err error) error {
	if err == nil {
		_, e := q.ExecContext(
			ctx,
			fmt.Sprintf(`UPDATE %s.%s SET status=?, ended_at=NOW() WHERE id=?`, mysqlDBName, mysqlEmailEntryTableName),
			sendmail.EmailEntry_SUCCESS,
			id,
		)
		return e
	} else {
		_, e := q.ExecContext(
			ctx,
			fmt.Sprintf(`UPDATE %s.%s SET status=?, ended_at=NOW(), failed_reason=? WHERE id=?`, mysqlDBName, mysqlEmailEntryTableName),
			sendmail.EmailEntry_FAILED,
			err.Error(),
			id,
		)
		return e
	}
}

func selectEmailEntry(ctx context.Context, q sqlh.Queryer, id int64) (*sendmail.EmailEntry, error) {
	return scanEmailEntry(q.QueryRowContext(ctx, selectEmailEntrySql("id=?"), id))
}

func selectEmailEntries(ctx context.Context, q sqlh.Queryer) (*sendmail.EmailEntries, error) {
	rows, err := q.QueryContext(ctx, selectEmailEntrySql("1=1"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := &sendmail.EmailEntries{
		Entries: []*sendmail.EmailEntry{},
	}
	for rows.Next() {
		entry, err := scanEmailEntry(rows)
		if err != nil {
			return nil, err
		}
		entries.Entries = append(entries.Entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil

}

func selectEmailEntrySql(where string) string {
	return fmt.Sprintf(`
	SELECT
		id,
		status,
		created_at,
		ended_at,
		failed_reason,
		to_addr,
		to_name,
		subject,
		content_type,
		content
	FROM %s.%s WHERE %s ORDER BY id DESC`, mysqlDBName, mysqlEmailEntryTableName, where)
}

type Scanable interface {
	Scan(dest ...interface{}) error
}

func scanEmailEntry(r Scanable) (*sendmail.EmailEntry, error) {
	entry := &sendmail.EmailEntry{
		Email: &sendmail.Email{},
	}
	err := r.Scan(
		&entry.Id,
		&entry.Status,
		&entry.CreatedAt,
		&entry.EndedAt,
		&entry.FailedReason,
		&entry.Email.ToAddr,
		&entry.Email.ToName,
		&entry.Email.Subject,
		&entry.Email.ContentType,
		&entry.Email.Content,
	)
	if err != nil {
		return nil, err
	}
	return entry, nil
}
