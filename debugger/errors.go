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
	mod string
	pc  uint64
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

func (e *simulateException) String() string {
	if e.mod == "" {
		return fmt.Sprintf("pc: %016X", e.pc)
	}
	return fmt.Sprintf("module: %s, offset: %08X", e.mod, e.pc)
}

func (e *simulateException) Context() Context {
	return e.ctx
}

func (e *InterruptException) Error() string {
	return fmt.Sprintf("[Interrupt] %s, intno: %d", &e.simulateException, e.intno)
}

func (e *InterruptException) Number() uint64 {
	return e.intno
}

func (e *InvalidInstructionException) Error() string {
	return fmt.Sprintf("[InvalidInstruction] %s", &e.simulateException)
}

func (e *InvalidMemoryException) Error() string {
	return fmt.Sprintf("[InvalidMemory] %s, type: %v, addr: %016X, size: %d, value: %d", &e.simulateException, e.typ, e.addr, e.size, e.value)
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
	return fmt.Sprintf("[Panic] %s, panic: %v", &e.simulateException, e.v)
}

func (e *PanicException) Panic() any {
	return e.v
}

func initException(ctx Context) simulateException {
	pc, _ := ctx.RegRead(ctx.PC())
	var mod string
	if m, err := ctx.Debugger().FindModuleByAddr(pc); err == nil {
		mod = m.Name()
		pc -= m.BaseAddr()
	}
	return simulateException{
		ctx: ctx,
		mod: mod,
		pc:  pc,
	}
}

func NewInterruptException(ctx Context, intno uint64) SimulateException {
	return &InterruptException{
		simulateException: initException(ctx),
		intno:             intno,
	}
}

func NewInvalidInstructionException(ctx Context) SimulateException {
	return &InvalidInstructionException{
		simulateException: initException(ctx),
	}
}

func NewInvalidMemoryException(ctx Context, typ emulator.HookType, addr, size, value uint64) SimulateException {
	return &InvalidMemoryException{
		simulateException: initException(ctx),
		typ:               typ,
		addr:              addr,
		size:              size,
		value:             value,
	}
}

func NewPanicException(ctx Context, v any) SimulateException {
	return &PanicException{
		simulateException: initException(ctx),
		v:                 v,
	}
}
