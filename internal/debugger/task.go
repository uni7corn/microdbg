package debugger

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
)

type task interface {
	debugger.Task
	appendRelease(func() error)
	contextSave() error
	contextRestore() error
	hasChange() bool
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
	go tm.start(dbg)
}

func (tm *taskManager) dtor() {
	close(tm.closed)
	for _, release := range tm.releases {
		release()
	}
}

func (tm *taskManager) start(dbg Debugger) {
	stopped := make(chan struct{})
	defer close(stopped)
	tm.releases = append(tm.releases, func() {
		<-stopped
	})
	go tm.collection()
	select {
	case <-tm.closed:
		return
	case task := <-tm.dispatch:
		err := task.contextRestore()
		if err != nil {
			tm.emuerr = err
			task.CancelCause(err)
			return
		}
		emu := dbg.Emulator()
		pc, err := emu.RegRead(dbg.PC())
		if err != nil {
			tm.emuerr = err
			task.CancelCause(err)
			return
		}
		tm.current = task
		go func() {
			tm.loop()
			emu.Stop()
		}()
		err = emu.Start(pc, math.MaxUint64)
		if err == nil {
			err = debugger.ErrEmulatorStop
		} else {
			err = debugger.NewPanicException(newGlobalContext(dbg), err)
		}
		tm.emuerr = err
		tm.current = nil
	}
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
			if task.Status() == debugger.TaskStatus_Done {
				goto wait
			}
			if task == tm.current && !task.hasChange() {
				break
			}
			tm.current = task
			err := task.contextRestore()
			if err != nil {
				task.CancelCause(err)
				goto wait
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
			n := len(contexts) - 1
			if n == -1 {
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
	err := task.contextSave()
	if err != nil {
		task.CancelCause(err)
	} else if tm.suspendTask() {
		task.async(fn)
		tm.resumeTask()
	}
	fmt.Print()
}

func (tm *taskManager) syncTask(fn func(debugger.Task)) {
	task := tm.current
	if task == nil {
		return
	}
	err := task.contextSave()
	if err != nil {
		task.CancelCause(err)
		return
	}
	defer func() {
		if ex := recover(); ex != nil {
			task.CancelCause(debugger.NewPanicException(task.Context(), ex))
		}
	}()
	tm.hasSync = true
	fn(task)
	tm.hasSync = false
	if !task.hasChange() {
		return
	} else if err = task.contextRestore(); err == nil {
		return
	}
	task.CancelCause(err)
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

func (dbg *Dbg) CreateTask(ctx context.Context) (debugger.Task, error) {
	tc, err := dbg.taskManager.allocTaskContext()
	if err != nil {
		return nil, err
	}
	return newTask(ctx, tc, dbg.impl)
}

func (dbg *Dbg) CallTaskOf(t debugger.Task, addr uint64) error {
	ctrl, err := dbg.impl.TaskControl(t, addr)
	if err != nil {
		return err
	}
	t.(task).appendRelease(ctrl.Close)
	return nil
}
