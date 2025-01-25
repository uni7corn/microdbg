package debugger

import (
	"context"
	"math"
	"runtime/debug"
	"sync/atomic"
	"unsafe"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
)

type task interface {
	debugger.Task
	isChange() bool
	appendRelease(func() error)
	contextSave() error
	contextRestore() error
	async(func(debugger.Task))
}

type taskContext struct {
	releases []func()
	ctx      emulator.Context
	stack    uint64
}

type taskManager struct {
	releases []func()
	closed   chan struct{}
	contexts chan *taskContext
	dispatch chan task
	suspend  chan struct{}
	resume   chan struct{}
	exec     chan func()
	hasSync  bool
	id       int64
	main     *mainTask
	current  task
	emuerr   error
}

func (tm *taskManager) ctor(dbg Debugger) {
	tm.closed = make(chan struct{})
	tm.contexts = make(chan *taskContext)
	tm.dispatch = make(chan task)
	tm.suspend = make(chan struct{})
	tm.resume = make(chan struct{})
	tm.exec = make(chan func())
	tm.start(dbg)
}

func (tm *taskManager) dtor() {
	close(tm.closed)
	for _, release := range tm.releases {
		release()
	}
}

func (tm *taskManager) start(dbg Debugger) {
	main, err := newMain(dbg)
	if err != nil {
		panic(err)
	}
	tm.main = main
	hook, err := dbg.AddControl(func(ctx debugger.Context, data any) { main.Close() }, nil)
	if err != nil {
		panic(err)
	}
	main.appendRelease(hook.Close)
	main.reset(context.TODO(), dbg)
	tm.current = main
	addr := hook.Addr()
	go func() {
		stopped := make(chan struct{})
		defer close(stopped)
		tm.releases = append(tm.releases, func() {
			<-stopped
		})
		go tm.collection()
		emu := dbg.Emulator()
		go func() {
			tm.loop()
			emu.Stop()
		}()
		err := emu.Start(addr, math.MaxUint64)
		if err == nil {
			err = debugger.ErrEmulatorStop
		} else {
			err = debugger.NewPanicException(newGlobalContext(dbg), err, nil)
		}
		tm.emuerr = err
		if tm.current != nil {
			tm.current.CancelCause(err)
		}
		tm.current = nil
		main.release()
	}()
}

func (tm *taskManager) loop() {
	for {
		select {
		case <-tm.closed:
			return
		case <-tm.suspend:
		}
	wait:
		select {
		case <-tm.closed:
			goto exit
		case task := <-tm.dispatch:
			if tm.current != task || task.isChange() {
				tm.current = task
				err := task.contextRestore()
				if err != nil {
					task.CancelCause(err)
					goto wait
				}
			}
		}
	exit:
		tm.resume <- struct{}{}
	}
}

func (tm *taskManager) collection() {
	var current *taskContext
	var contexts []*taskContext
	tm.releases = append(tm.releases, func() {
		if current != nil {
			current.dtor()
		}
		for _, ctx := range contexts {
			ctx.dtor()
		}
	})
	for {
		select {
		case <-tm.closed:
			return
		case tm.contexts <- current:
			if n := len(contexts) - 1; n == -1 {
				current = nil
			} else {
				current = contexts[n]
				contexts = contexts[:n]
			}
		case ctx := <-tm.contexts:
			if current == nil {
				current = ctx
			} else {
				contexts = append(contexts, ctx)
			}
		}
	}
}

func (tm *taskManager) suspendTask() bool {
	select {
	case <-tm.closed:
		return false
	case tm.suspend <- struct{}{}:
		return true
	}
}

func (tm *taskManager) resumeTask() {
	for {
		select {
		case <-tm.resume:
			return
		case fn := <-tm.exec:
			fn()
		}
	}
}

func (tm *taskManager) taskID() int {
	return int(atomic.AddInt64(&tm.id, 1))
}

func (tm *taskManager) newTaskContext(dbg Debugger) (*taskContext, error) {
	ctx, err := dbg.Emulator().ContextAlloc()
	if err != nil {
		return nil, err
	}
	stackAddr, err := dbg.InitStack()
	if err != nil {
		ctx.Close()
		return nil, err
	}
	return &taskContext{
		releases: []func(){
			func() { ctx.Close() },
			func() { dbg.CloseStack(stackAddr) },
		},
		ctx: ctx, stack: stackAddr,
	}, nil
}

func (tm *taskManager) allocTaskContext() (*taskContext, error) {
	select {
	case <-tm.closed:
		return nil, debugger.ErrEmulatorStop
	case ctx := <-tm.contexts:
		return ctx, nil
	}
}

func (tm *taskManager) freeTaskContext(ctx *taskContext) {
	if len(ctx.releases) == 1 {
		ctx.dtor()
		return
	}
	select {
	case <-tm.closed:
		ctx.dtor()
	case tm.contexts <- ctx:
	}
}

func (tm *taskManager) getMainTask(ctx context.Context, dbg Debugger) (debugger.Task, error) {
	if !atomic.CompareAndSwapUintptr((*uintptr)(unsafe.Pointer(&tm.main.status)), uintptr(debugger.TaskStatus_Close), uintptr(debugger.TaskStatus_Pending)) {
		return nil, debugger.TaskStatus_Running
	}
	tm.main.reset(ctx, dbg)
	return tm.main, nil
}

func (tm *taskManager) createTask(ctx context.Context, dbg Debugger) (debugger.Task, error) {
	tc, err := tm.allocTaskContext()
	if err != nil {
		return nil, err
	}
	return newTask(ctx, tc, dbg)
}

func (tm *taskManager) runTask(task task) {
	if tm.emuerr != nil {
		task.CancelCause(tm.emuerr)
		return
	}
	select {
	case <-tm.closed:
	case <-task.Done():
	case tm.dispatch <- task:
	}
}

func (tm *taskManager) asyncTask(fn func(debugger.Task)) {
	task := tm.current
	if task == nil {
		return
	}
	call := task.Status() < debugger.TaskStatus_Done
	if !call {
	} else if err := task.contextSave(); err != nil {
		task.CancelCause(err)
		call = false
	}
	if !tm.suspendTask() {
		return
	} else if call {
		task.async(fn)
	}
	tm.resumeTask()
}

func (tm *taskManager) syncTask(fn func(debugger.Task)) {
	task := tm.current
	if task == nil {
		return
	}
	if task.Status() >= debugger.TaskStatus_Done {
	} else if err := task.contextSave(); err != nil {
		task.CancelCause(err)
	} else {
		defer func() {
			if ex := recover(); ex != nil {
				task.CancelCause(debugger.NewPanicException(task.Context(), ex, debug.Stack()))
			}
		}()
		tm.hasSync = true
		fn(task)
		tm.hasSync = false
		if task.Status() >= debugger.TaskStatus_Done {
		} else if !task.isChange() {
			return
		} else if err = task.contextRestore(); err != nil {
			task.CancelCause(err)
		} else {
			return
		}
	}
	if tm.suspendTask() {
		tm.resumeTask()
	}
}

func (tm *taskManager) mainThreadRun(fn func()) {
	if tm.current == nil || tm.hasSync {
		fn()
		return
	}
	done := make(chan struct{})
	select {
	case <-tm.closed:
		fn()
	case tm.exec <- func() { fn(); close(done) }:
		<-done
	}
}

func (tc *taskContext) dtor() {
	for _, release := range tc.releases {
		release()
	}
}

func (tc *taskContext) clone() (*taskContext, error) {
	newCtx, err := tc.ctx.Clone()
	if err != nil {
		return nil, err
	}
	return &taskContext{
		releases: []func(){func() { newCtx.Close() }},
		ctx:      newCtx, stack: tc.stack,
	}, nil
}

func (dbg *Dbg) GetMainTask(ctx context.Context) (debugger.Task, error) {
	return dbg.taskManager.getMainTask(ctx, dbg.impl)
}

func (dbg *Dbg) CreateTask(ctx context.Context) (debugger.Task, error) {
	return dbg.taskManager.createTask(ctx, dbg.impl)
}

func (dbg *Dbg) CallTaskOf(t debugger.Task, addr uint64) error {
	ctrl, err := dbg.impl.TaskControl(t, addr)
	if err != nil {
		return err
	}
	t.(task).appendRelease(ctrl.Close)
	return nil
}
