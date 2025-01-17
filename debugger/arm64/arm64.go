package arm64

import (
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	internal "github.com/wnxd/microdbg/internal/debugger/arm64"
)

var _ = debugger.Register(emulator.ARCH_ARM64, internal.NewArm64Debugger)
