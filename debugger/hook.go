package debugger

import (
	"io"

	"github.com/wnxd/microdbg/emulator"
)

type HookResult int

const (
	HookResult_Done HookResult = -1
	HookResult_Next HookResult = 0
)

type InterruptCallback = func(ctx Context, intno uint64, data any) HookResult
type InvalidCallback = func(ctx Context, data any) HookResult
type MemoryCallback = func(ctx Context, typ emulator.HookType, addr, size, value uint64, data any) HookResult
type CodeCallback = func(ctx Context, addr, size uint64, data any)
type ControlCallback = func(ctx Context, data any)

type HookManger interface {
	AddHook(typ emulator.HookType, callback any, data any, begin, end uint64) (HookHandler, error)
	AddControl(callback ControlCallback, data any) (ControlHandler, error)
}

type HookHandler interface {
	io.Closer
	Type() emulator.HookType
}

type ControlHandler interface {
	io.Closer
	Addr() uint64
}
