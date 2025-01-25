package debugger

import (
	"io"

	"github.com/wnxd/microdbg/emulator"
)

type Debugger interface {
	io.Closer
	DebuggerInfo
	MemoryManager
	HookManger
	TaskManager
	ModuleManager
	FileManager
}

type DebuggerInfo interface {
	Emulator() emulator.Emulator
	Arch() emulator.Arch
	PointerSize() uint64
	StackSize() uint64
	StackAlign() uint64
}

func New(emu emulator.Emulator) (Debugger, error) {
	if ctor, ok := dbgMap[emu.Arch()]; ok {
		return ctor(emu)
	}
	return nil, emulator.ErrArchUnsupported
}
