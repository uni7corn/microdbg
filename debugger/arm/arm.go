package arm

import (
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	internal "github.com/wnxd/microdbg/internal/debugger/arm"
)

var _ = debugger.Register(emulator.ARCH_ARM, internal.NewArmDebugger)
