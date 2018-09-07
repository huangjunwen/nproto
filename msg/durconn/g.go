package durconn

import (
	"github.com/huangjunwen/nproto/util"
	"github.com/nats-io/go-nats-streaming"
)

var (
	stanConnect                      = stan.Connect
	cfh         util.ControlFlowHook = util.ProdControlFlowHook{}
)
