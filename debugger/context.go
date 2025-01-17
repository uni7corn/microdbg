package debugger

import (
	"github.com/wnxd/microdbg/emulator"
)

type Context interface {
	Debugger() Debugger
	PC() emulator.Reg
	SP() emulator.Reg
	TaskID() int
	ParentID() int
	TaskFork() (Task, error)
	RegisterContext
	GetArgs(calling Calling) (Args, error)
	ArgExtract(calling Calling, args ...any) error
	ArgWrite(calling Calling, args ...any) error
	RetExtract(val any) error
	RetWrite(val any) error
	Return() error
	Goto(addr uint64) error
	MemoryContext
	StorageContext
}

type RegisterContext interface {
	StackAlloc(size uint64) (emulator.Pointer, error)
	StackFree(size uint64) error
	emulator.RegisterContext
}

type MemoryContext interface {
	ToPointer(addr uint64) emulator.Pointer
}

type StorageContext interface {
	LocalStore(key, val any)
	LocalLoad(key any) (any, bool)
	LocalDelete(key any)
}
