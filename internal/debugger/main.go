package debugger

import (
	"context"

	"github.com/wnxd/microdbg/debugger"
)

type mainTask struct {
	runner
	mainCtx    context.Context
	mainCancel context.CancelCauseFunc
}

func newMain(dbg Debugger) (*mainTask, error) {
	tc, err := dbg.newTaskContext(dbg)
	if err != nil {
		return nil, err
	}
	err = tc.ctx.Save()
	if err != nil {
		dbg.freeTaskContext(tc)
		return nil, err
	}
	task := new(mainTask)
	task.dbg = dbg
	task.id = dbg.taskID()
	task.taskCtx = tc
	task.mainCtx, task.mainCancel = context.WithCancelCause(context.TODO())
	return task, nil
}

func (m *mainTask) reset(ctx context.Context, dbg Debugger) {
	m.taskCtx.ctx.RegWrite(dbg.SP(), m.taskCtx.stack)
	m.change = true
	m.status = debugger.TaskStatus_Pending
	m.ctx, m.cancel = context.WithCancelCause(ctx)
	ch := make(chan func(debugger.Task))
	m.send = ch
	go m.loop(m.ctx, ch)
}

func (m *mainTask) release() error {
	m.Close()
	m.mainCancel(nil)
	m.dbg.freeTaskContext(m.taskCtx)
	return nil
}

func (m *mainTask) Close() error {
	m.storage.Clear()
	for i := len(m.releases) - 1; i >= 0; i-- {
		m.releases[i]()
	}
	m.releases = nil
	m.CancelCause(nil)
	return nil
}

func (m *mainTask) Fork() (debugger.Task, error) {
	newCtx, err := m.taskCtx.clone()
	if err != nil {
		return nil, err
	}
	task := newRunner(m.mainCtx, newCtx, m.dbg)
	task.pid = m.dbg.taskID()
	return task, nil
}
