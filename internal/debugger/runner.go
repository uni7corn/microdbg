package debugger

import (
	"context"
	"errors"
	"unsafe"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
)

type runner struct {
	baseContext[*runner]
	releases []func() error
	ctx      context.Context
	cancel   context.CancelCauseFunc
	id, pid  int
	taskCtx  *taskContext
	send     chan<- func(debugger.Task)
	change   bool
	regs     map[emulator.Reg]uint64
	status   debugger.TaskStatus
}

func newTask(ctx context.Context, tc *taskContext, dbg Debugger) (task, error) {
	var err error
	if tc == nil {
		tc, err = dbg.newTaskContext(dbg)
		if err != nil {
			return nil, err
		}
	}
	err = tc.ctx.Save()
	if err != nil {
		dbg.freeTaskContext(tc)
		return nil, err
	}
	tc.ctx.RegWrite(dbg.SP(), tc.stack)
	return newRunner(ctx, tc, dbg), nil
}

func newRunner(ctx context.Context, tc *taskContext, dbg Debugger) *runner {
	task := &runner{baseContext: baseContext[*runner]{dbg: dbg},
		id: dbg.taskID(), taskCtx: tc,
		regs: make(map[emulator.Reg]uint64),
	}
	task.ctx, task.cancel = context.WithCancelCause(ctx)
	ch := make(chan func(debugger.Task))
	task.send = ch
	go task.loop(ch)
	return task
}

func (r *runner) Close() error {
	for i := len(r.releases) - 1; i >= 0; i-- {
		r.releases[i]()
	}
	r.releases = nil
	r.CancelCause(nil)
	r.dbg.freeTaskContext(r.taskCtx)
	return nil
}

func (r *runner) ID() int {
	return r.id
}

func (r *runner) ParentID() int {
	return r.pid
}

func (r *runner) Status() debugger.TaskStatus {
	return r.status
}

func (r *runner) Context() debugger.Context {
	return r
}

func (r *runner) Run() error {
	if r.status != debugger.TaskStatus_Pending {
		return r.status
	}
	go r.dbg.runTask(r)
	return nil
}

func (r *runner) SyncRun() error {
	err := r.Run()
	if err != nil {
		return err
	}
	<-r.Done()
	return r.Err()
}

func (r *runner) Done() <-chan struct{} {
	return r.ctx.Done()
}

func (r *runner) Err() error {
	err := context.Cause(r.ctx)
	if errors.Is(err, debugger.TaskStatus_Done) {
		err = nil
	}
	return err
}

func (r *runner) CancelCause(cause error) {
	r.status = debugger.TaskStatus_Done
	r.cancel(cause)
}

func (r *runner) Fork() (debugger.Task, error) {
	newCtx, err := r.taskCtx.clone()
	if err != nil {
		return nil, err
	}
	task := newRunner(r.ctx, newCtx, r.dbg)
	task.pid = r.id
	return task, nil
}

func (r *runner) appendRelease(f func() error) {
	r.releases = append(r.releases, f)
}

func (r *runner) contextSave() error {
	r.status = debugger.TaskStatus_Running
	return r.taskCtx.ctx.Save()
}

func (r *runner) contextRestore() error {
	defer clear(r.regs)
	if r.change {
		r.change = false
		switch len(r.regs) {
		case 0:
		case 1:
			for reg, val := range r.regs {
				r.taskCtx.ctx.RegWrite(reg, val)
			}
		default:
			regs := make([]emulator.Reg, 0, len(r.regs))
			vals := make([]uint64, 0, len(r.regs))
			for reg, val := range r.regs {
				regs = append(regs, reg)
				vals = append(vals, val)
			}
			r.taskCtx.ctx.RegWriteBatch(regs, vals)
		}
		return r.taskCtx.ctx.Restore()
	}
	return nil
}

func (r *runner) hasChange() bool {
	return r.change
}

func (r *runner) async(fn func(debugger.Task)) {
	select {
	case <-r.ctx.Done():
	case r.send <- fn:
	}
}

func (r *runner) loop(recv <-chan func(debugger.Task)) {
	for {
		select {
		case <-r.ctx.Done():
			return
		case fn := <-recv:
			r.handle(fn)
		}
	}
}

func (r *runner) handle(fn func(debugger.Task)) {
	defer func() {
		if ex := recover(); ex != nil {
			r.CancelCause(debugger.NewPanicException(r.Context(), ex))
		}
	}()
	fn(r)
	r.dbg.runTask(r)
}

func (r *runner) TaskID() int {
	return r.id
}

func (r *runner) TaskFork() (debugger.Task, error) {
	return r.Fork()
}

func (r *runner) RegRead(reg emulator.Reg) (uint64, error) {
	if val, ok := r.regs[reg]; ok {
		return val, nil
	}
	val, err := r.taskCtx.ctx.RegRead(reg)
	if err == nil {
		r.regs[reg] = val
	}
	return val, err
}

func (r *runner) RegWrite(reg emulator.Reg, value uint64) error {
	r.change = true
	r.regs[reg] = value
	return nil
}

func (r *runner) RegReadPtr(reg emulator.Reg, ptr unsafe.Pointer) error {
	return r.taskCtx.ctx.RegReadPtr(reg, ptr)
}

func (r *runner) RegWritePtr(reg emulator.Reg, ptr unsafe.Pointer) error {
	return r.taskCtx.ctx.RegWritePtr(reg, ptr)
}

func (r *runner) RegReadBatch(regs ...emulator.Reg) ([]uint64, error) {
	vals := make([]uint64, len(regs))
	for i, reg := range regs {
		val, ok := r.regs[reg]
		if !ok {
			reads, err := r.taskCtx.ctx.RegReadBatch(regs[i:]...)
			if err != nil {
				return nil, err
			}
			for n, reg := range regs[i:] {
				r.regs[reg] = reads[n]
			}
			copy(vals[i:], reads)
			break
		}
		vals[i] = val
	}
	return vals, nil
}

func (r *runner) RegWriteBatch(regs []emulator.Reg, vals []uint64) error {
	if len(regs) != len(vals) {
		return debugger.ErrArgumentInvalid
	}
	r.change = true
	for i, reg := range regs {
		r.regs[reg] = vals[i]
	}
	return nil
}
