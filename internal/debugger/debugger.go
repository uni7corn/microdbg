package debugger

import (
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

type Dbg struct {
	impl Debugger
	emu  emulator.Emulator
	memoryManager
	hookManger
	fileManager
	moduleManager
	taskManager
}

func (dbg *Dbg) Init(impl Debugger, emu emulator.Emulator) error {
	dbg.impl = impl
	dbg.emu = emu
	dbg.memoryManager.ctor()
	dbg.hookManger.ctor(dbg.impl)
	dbg.fileManager.ctor()
	dbg.moduleManager.ctor()
	dbg.taskManager.ctor(dbg.impl)
	return nil
}

func (dbg *Dbg) Close() error {
	dbg.taskManager.dtor()
	dbg.moduleManager.dtor()
	dbg.fileManager.dtor()
	dbg.hookManger.dtor()
	dbg.memoryManager.dtor(dbg.impl)
	return nil
}

func (dbg *Dbg) Emulator() emulator.Emulator {
	return dbg.emu
}
