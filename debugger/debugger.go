package debugger

import (
	"io"

	"github.com/wnxd/microdbg/emulator"
)

type Debugger interface {
	io.Closer
	Emulator() emulator.Emulator
	MemoryManager
	HookManger
	TaskManager
	ModuleManager
	FileManager
}

func New(emu emulator.Emulator) (Debugger, error) {
	if ctor, ok := dbgMap[emu.Arch()]; ok {
		return ctor(emu)
	}
	return nil, emulator.ErrArchUnsupported
}
