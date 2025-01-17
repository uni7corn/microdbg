package debugger

import (
	"unsafe"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
)

type Debugger interface {
	debugger.Debugger
	PointerSize() uint64
	StackAlign() uint64
	PC() emulator.Reg
	SP() emulator.Reg
	Args(debugger.RegisterContext, debugger.Calling) (debugger.Args, error)
	ArgWrite(debugger.RegisterContext, debugger.Calling, ...any) error
	RetExtract(debugger.RegisterContext, any) error
	RetWrite(debugger.RegisterContext, any) error
	Return(debugger.RegisterContext) error
	InitStack() (uint64, error)
	CloseStack(uint64) error
	TaskControl(debugger.Task, uint64) (debugger.ControlHandler, error)
	taskID() int
	newTaskContext(Debugger) (*taskContext, error)
	allocTaskContext() (*taskContext, error)
	freeTaskContext(*taskContext)
	runTask(task)
	asyncTask(func(debugger.Task))
	syncTask(func(debugger.Task))
	mainThreadRun(func())
}

type Dbg[Impl Debugger] struct {
	emu emulator.Emulator
	memoryManager
	hookManger
	fileManager
	moduleManager
	taskManager
}

func (dbg *Dbg[Impl]) Init(emu emulator.Emulator) error {
	impl := dbg.impl()
	dbg.emu = emu
	dbg.memoryManager.ctor()
	dbg.hookManger.ctor(impl)
	dbg.fileManager.ctor()
	dbg.moduleManager.ctor()
	dbg.taskManager.ctor(impl)
	return nil
}

func (dbg *Dbg[Impl]) Close() error {
	impl := dbg.impl()
	dbg.taskManager.dtor()
	dbg.moduleManager.dtor()
	dbg.fileManager.dtor()
	dbg.hookManger.dtor()
	dbg.memoryManager.dtor(impl)
	return nil
}

func (dbg *Dbg[Impl]) impl() Debugger {
	return *(*Impl)(unsafe.Pointer(&dbg))
}

func (dbg *Dbg[Impl]) Emulator() emulator.Emulator {
	return dbg.emu
}
