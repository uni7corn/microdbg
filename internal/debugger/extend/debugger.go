package extend

import (
	"github.com/wnxd/microdbg/emulator"
	internal "github.com/wnxd/microdbg/internal/debugger"
)

type ExtendDebugger interface {
	internal.Debugger
	Init(internal.Debugger, emulator.Emulator) error
}
