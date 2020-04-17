package fulldump

import (
	"context"

	"github.com/huangjunwen/nproto/helpers/sqlh"
)

// Handler is used to dump content.
type Handler func(ctx context.Context, q sqlh.Queryer) error
