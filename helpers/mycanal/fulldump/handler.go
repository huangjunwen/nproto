package fulldump

import (
	"context"

	sqlh "github.com/huangjunwen/nproto/helpers/sql"
)

// Handler is used to dump content.
type Handler func(ctx context.Context, q sqlh.Queryer) error
