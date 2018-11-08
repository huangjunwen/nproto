package main

import (
	"context"
	"log"
	"time"

	"github.com/huangjunwen/nproto"
	"github.com/huangjunwen/nproto/npmsg"
	"github.com/huangjunwen/nproto/npmsg/durconn"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/nats-io/go-nats"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL, nats.MaxReconnects(-1))
	if err != nil {
		log.Panic(err)
	}
	defer nc.Close()

	dc, err := durconn.NewDurConn(nc, "test-cluster")
	if err != nil {
		log.Panic(err)
	}
	defer dc.Close()

	time.Sleep(time.Second)

	publisher := npmsg.NewMsgPublisher(dc, nil)
	subscriber := npmsg.NewMsgSubscriber(dc, nil)

	subject := "fib"

	err = subscriber.Subscribe(
		subject,
		"def",
		func() proto.Message {
			return &wrappers.UInt64Value{}
		},
		func(ctx context.Context, msg proto.Message) error {
			v := msg.(*wrappers.UInt64Value).Value
			subj := nproto.CurrMsgSubject(ctx)
			passthru := nproto.Passthru(ctx)
			log.Printf("Recv %d from %+q passthru=%v\n", v, subj, passthru)
			return nil
		},
	)
	if err != nil {
		log.Panic(err)
	}

	var prev, curr uint64 = 0, 1
	for {
		ctx := nproto.AddPassthru(context.Background(), "now", time.Now().String())
		err := publisher.Publish(ctx, subject, &wrappers.UInt64Value{
			Value: curr,
		})
		time.Sleep(time.Second)
		if err != nil {
			continue
		}
		x := curr + prev
		prev = curr
		curr = x
	}
}
