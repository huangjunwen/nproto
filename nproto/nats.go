package nproto

import (
	"context"
	"errors"
	"fmt"
	"sync"

	//"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
)

var (
	// Default convertor for service name and nats subject name.
	DefaultSvcSubjNameConvertor = func(name string) string { return fmt.Sprint("svc.%s", name) }
	// Default queue group.
	DefaultGroup = "def"
)

var (
	ErrNilConn      = errors.New("nproto: conn is nil")
	ErrServerClosed = errors.New("nproto: server closed")
)

type NatsConn interface {
	PublishRequest(subj, reply string, data []byte) error
	RequestWithContext(ctx context.Context, subj string, data []byte) (*nats.Msg, error)
	Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
	QueueSubscribe(subj, queue string, cb nats.MsgHandler) (*nats.Subscription, error)
}

type NatsRPCServer struct {
	nameConv func(string) string // svcName -> subjName
	group    string              // server group
	mws      []RPCMiddleware     // middleware
	mu       sync.RWMutex
	conn     NatsConn                         // nil if closed
	subs     map[string]*nats.Subscription    // svcName -> Subscription
	methods  map[string]map[string]RPCHandler // svcName -> methodName -> handler
}

type NatsRPCServerOption func(*NatsRPCServer) error

var (
	_ NatsConn = (*nats.Conn)(nil)
)

func NewNatsRPCServer(conn NatsConn, opts ...NatsRPCServerOption) (*NatsRPCServer, error) {
	if conn == nil {
		return nil, ErrNilConn
	}

	server := &NatsRPCServer{
		nameConv: DefaultSvcSubjNameConvertor,
		group:    DefaultGroup,
		conn:     conn,
		subs:     make(map[string]*nats.Subscription),
	}
	for _, opt := range opts {
		if err := opt(server); err != nil {
			return nil, err
		}
	}

	return server, nil
}

// RegistSvc regist a service with given name and methods.
func (server *NatsRPCServer) RegistSvc(svcName string, methods map[string]RPCHandler) error {
	// First (read) lock: get conn and check closed?
	server.mu.RLock()
	conn := server.conn
	server.mu.RUnlock()
	if conn == nil {
		return ErrServerClosed
	}

	// Decorate methods.
	decMethods := make(map[string]RPCHandler)
	for methodName, handler := range methods {
		for i := len(server.mws) - 1; i >= 0; i-- {
			handler = server.mws[i](handler)
		}
		decMethods[methodName] = handler
	}

	// Subscribe.
	subj := server.nameConv(svcName) + ".>"
	sub, err := server.conn.QueueSubscribe(subj, server.group, server.msgHandler(decMethods))
	if err != nil {
		return err
	}

	// Second (write) lock: replace subscriptions and methods
	server.mu.Lock()
	conn = server.conn
	oldSub := server.subs[svcName]
	if conn != nil {
		server.subs[svcName] = sub
		server.methods[svcName] = decMethods
	}
	server.mu.Unlock()

	// If there is an old subscription, release it.
	if oldSub != nil {
		oldSub.Unsubscribe()
	}

	// If conn is closed between two locks, release sub too.
	if conn == nil {
		sub.Unsubscribe()
		return ErrServerClosed
	}

	return nil

}

// DeregistSvc deregist a service from the server.
func (server *NatsRPCServer) DeregistSvc(svcName string) error {
	// Pop svcName.
	server.mu.Lock()
	sub := server.subs[svcName]
	delete(server.subs, svcName)
	delete(server.methods, svcName)
	server.mu.Unlock()

	// Unsubscribe.
	if sub != nil {
		return sub.Unsubscribe()
	}
	return nil
}

// Close the server and release all subscriptions on it.
func (server *NatsRPCServer) Close() error {
	// Set conn to nil to indicate close.
	server.mu.Lock()
	conn := server.conn
	subs := server.subs
	server.conn = nil
	server.subs = nil
	server.methods = nil
	server.mu.Unlock()

	// Release subscriptions.
	if len(subs) != 0 {
		for _, sub := range subs {
			sub.Unsubscribe()
		}
	}

	// Multiple calls to Close.
	if conn == nil {
		return ErrServerClosed
	}
	return nil
}

func (server *NatsRPCServer) msgHandler(methods map[string]RPCHandler) nats.MsgHandler {
	return func(msg *nats.Msg) {

	}
}

// UseSvcSubjNameConvertor sets the convertor to convert service name to subject name.
func UseSvcSubjNameConvertor(nameConv func(string) string) NatsRPCServerOption {
	return func(server *NatsRPCServer) error {
		if nameConv == nil {
			nameConv = DefaultSvcSubjNameConvertor
		}
		server.nameConv = nameConv
		return nil
	}
}

// UseGroup sets the group name for subscriptions. Note that NatsRPCServer
// always makes QueueSubscribe.
func UseGroup(group string) NatsRPCServerOption {
	return func(server *NatsRPCServer) error {
		if group == "" {
			group = DefaultGroup
		}
		server.group = group
		return nil
	}
}

// UseMiddleware inject a middleware to the server.
func UseMiddleware(mw RPCMiddleware) NatsRPCServerOption {
	return func(server *NatsRPCServer) error {
		if mw != nil {
			server.mws = append(server.mws, mw)
		}
		return nil
	}
}
