package pb

import (
	"fmt"
)

var (
	_ error = (*RPCErrorReply)(nil)
)

// Error implements error interface.
func (e *RPCErrorReply) Error() string {
	return fmt.Sprintf("Code=%d Msg=%+q", e.Code, e.Message)
}
