package testutil

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"

	"github.com/huangjunwen/nproto/util"
)

// TestControlFlowHook is used in testing.
type TestControlFlowHook struct {
	captureC chan *GoroutineController
	wg       sync.WaitGroup
	mu       sync.Mutex
	ctrls    map[uint64]*GoroutineController // gid -> GoroutineController
}

// GoroutineController is used to control suspensions of a go routine.
type GoroutineController struct {
	gid   uint64 // The controlled go routine's gid.
	label string // The controlled go routine's label.

	suspendC chan *suspension
	resumeC  chan interface{}

	mu            sync.Mutex
	curSuspension *suspension
}

type suspension struct {
	label   string
	payload interface{}
}

var (
	_ util.ControlFlowHook = (*TestControlFlowHook)(nil)
)

// NewTestControlFlowHook creates a new TestControlFlowHook.
func NewTestControlFlowHook() *TestControlFlowHook {
	return &TestControlFlowHook{
		captureC: make(chan *GoroutineController),
		ctrls:    make(map[uint64]*GoroutineController),
	}
}

// Go implements ControlFlowHook interface.
func (cfh *TestControlFlowHook) Go(label string, fn func()) {
	go func() {
		// Create a controller for current go routine.
		ctrl := &GoroutineController{
			gid:      getGID(),
			label:    label,
			suspendC: make(chan *suspension),
			resumeC:  make(chan interface{}),
		}

		// Add.
		cfh.wg.Add(1)
		cfh.mu.Lock()
		cfh.ctrls[ctrl.gid] = ctrl
		cfh.mu.Unlock()

		// Remove
		defer func() {
			cfh.mu.Lock()
			delete(cfh.ctrls, ctrl.gid)
			close(ctrl.suspendC)
			cfh.mu.Unlock()
			cfh.wg.Done()
		}()

		// Allow capture.
		go func() {
			cfh.captureC <- ctrl
		}()

		// First suspension.
		ctrl.suspend("", nil)

		fn()
	}()
}

// Capture captures a labeled go routine and return its controller (in suspension).
func (cfh *TestControlFlowHook) Capture(label string) *GoroutineController {
	for {
		ctrl := <-cfh.captureC
		if ctrl.label == label {
			// First expect.
			ctrl.Expect("")
			return ctrl
		}
		go func() {
			// Requeue.
			cfh.captureC <- ctrl
		}()
	}
}

// Suspend implements ControlFlowHook interface.
func (cfh *TestControlFlowHook) Suspend(label string, payload interface{}) interface{} {
	gid := getGID()

	cfh.mu.Lock()
	ctrl := cfh.ctrls[gid]
	cfh.mu.Unlock()

	if ctrl == nil {
		panic(fmt.Errorf("Suspending an unknown go routine %d. You must use TestControlFlowHook.Go to start a go routine.", gid))
	}
	return ctrl.suspend(label, payload)
}

// Wait for all go routines exist.
func (cfh *TestControlFlowHook) Wait() {
	cfh.wg.Wait()
}

func (ctrl *GoroutineController) suspend(label string, payload interface{}) interface{} {
	ctrl.suspendC <- &suspension{
		label:   label,
		payload: payload,
	}
	return <-ctrl.resumeC
}

// Expect waits the labeled suspension. If there is a suspension not resumed at the moment,
// it will be resumed with a nil. It panics if getting a suspension with different label.
func (ctrl *GoroutineController) Expect(label string) *GoroutineController {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	if ctrl.curSuspension != nil {
		// Resume with nil if there is one already.
		ctrl.resume(nil)
	}
	ctrl.expect(label)
	return ctrl
}

// Resume resumes current suspension. It panics if there is no suspension currently.
func (ctrl *GoroutineController) Resume(ret interface{}) *GoroutineController {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	ctrl.resume(ret)
	return ctrl
}

// Do something with current suspension's payload. It panics if there is no suspension currently.
func (ctrl *GoroutineController) Do(fn func(interface{})) *GoroutineController {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	ctrl.do(fn)
	return ctrl
}

func (ctrl *GoroutineController) resume(ret interface{}) {
	if ctrl.curSuspension == nil {
		panic(fmt.Errorf("Currently there is no suspension."))
	}
	ctrl.resumeC <- ret
	ctrl.curSuspension = nil
}

func (ctrl *GoroutineController) expect(label string) {
	if ctrl.curSuspension != nil {
		panic(fmt.Errorf("Previous suspension has not been resumed"))
	}
	s := <-ctrl.suspendC
	if s == nil {
		panic(fmt.Errorf("Expect label %+q but suspend channel is closed.", label))
	}
	if s.label != label {
		panic(fmt.Errorf("Expect label %+q but got %+q.", label, s.label))
	}
	ctrl.curSuspension = s
}

func (ctrl *GoroutineController) do(fn func(interface{})) {
	if ctrl.curSuspension == nil {
		panic(fmt.Errorf("Currently there is no suspension."))
	}
	fn(ctrl.curSuspension.payload)
}

var stackPrefix = []byte("goroutine ")

// getGID returns current goroutine's id. NOTE: For testing only.
// ref: https://blog.sgmansfield.com/2015/12/goroutine-ids/
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, stackPrefix)
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
