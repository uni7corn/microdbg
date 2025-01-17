package arm64

import (
	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/internal/debugger/arm64"
	"github.com/wnxd/microdbg/internal/debugger/extend"
)

type Arm64Dbg[D extend.ExtendDebugger] struct {
	arm64.Arm64Dbg[D]
}

func NewExtendDebugger[D extend.ExtendDebugger](emu emulator.Emulator) (D, error) {
	return arm64.NewExtendDebugger[D](emu)
}
