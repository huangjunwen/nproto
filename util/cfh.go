package util

// ControlFlowHook is used to add hooks in program.
type ControlFlowHook interface {
	// Go runs fn in a new go routine with a given label.
	// NOTE: Label does not need to be unique. It's only used to distinguish the purpose of fn.
	Go(label string, fn func())

	// Suspend pauses current execution and waits for resuming. And optionally exchanges some data.
	Suspend(label string, payload interface{}) interface{}
}

// ProdControlFlowHook is used in production.
type ProdControlFlowHook struct{}

// Go implements ControlFlowHook interface.
func (cfh ProdControlFlowHook) Go(label string, fn func()) {
	go fn()
}

// Suspend implements ControlFlowHook interface.
func (cfh ProdControlFlowHook) Suspend(label string, payload interface{}) interface{} {
	// Do nothing.
	return nil
}

var _ ControlFlowHook = ProdControlFlowHook{}
