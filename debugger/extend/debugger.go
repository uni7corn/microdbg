package extend

import (
	"github.com/wnxd/microdbg/emulator"
	internal "github.com/wnxd/microdbg/internal/debugger"
)

type ExtendDebugger interface {
	internal.Debugger
	ExtendInit(emu emulator.Emulator) error
}
