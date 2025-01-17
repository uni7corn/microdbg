package extend

import (
	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/internal/debugger"
)

type ExtendDebugger interface {
	debugger.Debugger
	ExtendInit(emu emulator.Emulator) error
}
