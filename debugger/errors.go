package debugger

import (
	"errors"
	"fmt"

	"github.com/wnxd/microdbg/emulator"
)

var (
	ErrContextInvalid     = errors.New("context invalid")
	ErrModuleNotFound     = errors.New("module not found")
	ErrSymbolNotFound     = errors.New("symbol not found")
	ErrHookCallbackType   = errors.New("hook callback type exception")
	ErrTaskInvalid        = errors.New("task invalid")
	ErrCallingUnsupported = errors.New("calling unsupported")
	ErrUnhandledException = errors.New("unhandled exception")
	ErrArgumentInvalid    = errors.New("argument invalid")
	ErrEmulatorStop       = errors.New("emulator stop")
	ErrAddressInvalid     = errors.New("address invalid")
	ErrNotImplemented     = errors.New("not implemented")
)

type SimulateException interface {
	error
	Context() Context
}

type simulateException struct {
	ctx Context
}

type InterruptException struct {
	simulateException
	intno uint64
}

type InvalidInstructionException struct {
	simulateException
}

type InvalidMemoryException struct {
	simulateException
	typ   emulator.HookType
	addr  uint64
	size  uint64
	value uint64
}

type PanicException struct {
	simulateException
	v any
}

func (e *simulateException) Error() string {
	return ErrUnhandledException.Error()
}

func (e *simulateException) Context() Context {
	return e.ctx
}

func (e *InterruptException) Error() string {
	pc, _ := e.ctx.RegRead(e.ctx.PC())
	return fmt.Sprintf("[Interrupt] pc: %016X, intno: %d", pc, e.intno)
}

func (e *InterruptException) Number() uint64 {
	return e.intno
}

func (e *InvalidInstructionException) Error() string {
	pc, _ := e.ctx.RegRead(e.ctx.PC())
	return fmt.Sprintf("[InvalidInstruction] pc: %016X", pc)
}

func (e *InvalidMemoryException) Error() string {
	pc, _ := e.ctx.RegRead(e.ctx.PC())
	return fmt.Sprintf("[InvalidMemory] pc: %016X, type: %v, addr: %016X, size: %d, value: %d", pc, e.typ, e.addr, e.size, e.value)
}

func (e *InvalidMemoryException) Type() emulator.HookType {
	return e.typ
}

func (e *InvalidMemoryException) Address() uint64 {
	return e.addr
}

func (e *InvalidMemoryException) Size() uint64 {
	return e.size
}

func (e *InvalidMemoryException) Value() uint64 {
	return e.value
}

func (e *PanicException) Error() string {
	pc, _ := e.ctx.RegRead(e.ctx.PC())
	return fmt.Sprintf("[Panic] pc: %016X, panic: %v", pc, e.v)
}

func (e *PanicException) Panic() any {
	return e.v
}

func NewInterruptException(ctx Context, intno uint64) SimulateException {
	return &InterruptException{
		simulateException: simulateException{
			ctx: ctx,
		},
		intno: intno,
	}
}

func NewInvalidInstructionException(ctx Context) SimulateException {
	return &InvalidInstructionException{
		simulateException: simulateException{
			ctx: ctx,
		},
	}
}

func NewInvalidMemoryException(ctx Context, typ emulator.HookType, addr, size, value uint64) SimulateException {
	return &InvalidMemoryException{
		simulateException: simulateException{
			ctx: ctx,
		},
		typ:   typ,
		addr:  addr,
		size:  size,
		value: value,
	}
}

func NewPanicException(ctx Context, v any) SimulateException {
	return &PanicException{
		simulateException: simulateException{
			ctx: ctx,
		},
		v: v,
	}
}
