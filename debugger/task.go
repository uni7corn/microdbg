package debugger

import (
	"context"
	"io"
)

type Calling int

const (
	Calling_Default = iota
	Calling_Cdecl
	Calling_Stdcall
	Calling_Fastcall
	Calling_ATPCS = Calling_Fastcall
)

type TaskStatus int

const (
	TaskStatus_Pending TaskStatus = iota
	TaskStatus_Running
	TaskStatus_Done
	TaskStatus_Close
)

type Task interface {
	io.Closer
	ID() int
	ParentID() int
	Status() TaskStatus
	Context() Context
	Run() error
	SyncRun() error
	Done() <-chan struct{}
	Err() error
	CancelCause(err error)
	Fork() (Task, error)
}

type TaskManager interface {
	GetMainTask(ctx context.Context) (Task, error)
	CreateTask(ctx context.Context) (Task, error)
	CallTaskOf(task Task, addr uint64) error
}

func (s TaskStatus) Error() string {
	switch s {
	case TaskStatus_Pending:
		return "pending"
	case TaskStatus_Running:
		return "running"
	case TaskStatus_Done:
		return "done"
	}
	return "unknown"
}
