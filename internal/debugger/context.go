package debugger

import (
	"errors"
	"os"
	"sync"
	"unsafe"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
)

type baseContext[Impl debugger.Context] struct {
	dbg     Debugger
	storage sync.Map
}

type globalContext struct {
	baseContext[*globalContext]
}

func newGlobalContext(dbg Debugger) debugger.Context {
	ctx := new(globalContext)
	ctx.dbg = dbg
	return ctx
}

func (bc *baseContext[Impl]) impl() debugger.Context {
	return *(*Impl)(unsafe.Pointer(&bc))
}

func (bc *baseContext[Impl]) Debugger() debugger.Debugger {
	return bc.dbg
}

func (bc *baseContext[Impl]) PC() emulator.Reg {
	return bc.dbg.PC()
}

func (bc *baseContext[Impl]) SP() emulator.Reg {
	return bc.dbg.SP()
}

func (bc *baseContext[Impl]) TaskID() int {
	return os.Getpid()
}

func (bc *baseContext[Impl]) ParentID() int {
	return os.Getppid()
}

func (bc *baseContext[Impl]) TaskFork() (debugger.Task, error) {
	return nil, errors.ErrUnsupported
}

func (bc *baseContext[Impl]) StackAlloc(size uint64) (emulator.Pointer, error) {
	ctx := bc.impl()
	size = debugger.Align(size, bc.dbg.StackAlign())
	stackAddr, err := ctx.RegRead(bc.SP())
	if err != nil {
		return emulator.Pointer{}, err
	}
	stackAddr -= size
	return bc.ToPointer(stackAddr), ctx.RegWrite(bc.SP(), stackAddr)
}

func (bc *baseContext[Impl]) StackFree(size uint64) error {
	ctx := bc.impl()
	size = debugger.Align(size, bc.dbg.StackAlign())
	stackAddr, err := ctx.RegRead(bc.SP())
	if err != nil {
		return err
	}
	stackAddr += size
	return ctx.RegWrite(bc.SP(), stackAddr)
}

func (bc *baseContext[Impl]) RegRead(reg emulator.Reg) (uint64, error) {
	return bc.dbg.Emulator().RegRead(reg)
}

func (bc *baseContext[Impl]) RegWrite(reg emulator.Reg, value uint64) error {
	return bc.dbg.Emulator().RegWrite(reg, value)
}

func (bc *baseContext[Impl]) RegReadPtr(reg emulator.Reg, ptr unsafe.Pointer) error {
	return bc.dbg.Emulator().RegReadPtr(reg, ptr)
}

func (bc *baseContext[Impl]) RegWritePtr(reg emulator.Reg, ptr unsafe.Pointer) error {
	return bc.dbg.Emulator().RegWritePtr(reg, ptr)
}

func (bc *baseContext[Impl]) RegReadBatch(regs ...emulator.Reg) ([]uint64, error) {
	return bc.dbg.Emulator().RegReadBatch(regs...)
}

func (bc *baseContext[Impl]) RegWriteBatch(regs []emulator.Reg, vals []uint64) error {
	return bc.dbg.Emulator().RegWriteBatch(regs, vals)
}

func (bc *baseContext[Impl]) GetArgs(calling debugger.Calling) (debugger.Args, error) {
	return bc.dbg.Args(bc.impl(), calling)
}

func (bc *baseContext[Impl]) ArgExtract(calling debugger.Calling, args ...any) error {
	va, err := bc.impl().GetArgs(calling)
	if err != nil {
		return err
	}
	return va.Extract(args...)
}

func (bc *baseContext[Impl]) ArgWrite(calling debugger.Calling, args ...any) error {
	return bc.dbg.ArgWrite(bc.impl(), calling, args...)
}

func (bc *baseContext[Impl]) RetExtract(val any) error {
	return bc.dbg.RetExtract(bc.impl(), val)
}

func (bc *baseContext[Impl]) RetWrite(val any) error {
	return bc.dbg.RetWrite(bc.impl(), val)
}

func (bc *baseContext[Impl]) Return() error {
	return bc.dbg.Return(bc.impl())
}

func (bc *baseContext[Impl]) Goto(addr uint64) error {
	ctx := bc.impl()
	return ctx.RegWrite(ctx.PC(), addr)
}

func (bc *baseContext[Impl]) ToPointer(addr uint64) emulator.Pointer {
	return emulator.ToPointer(bc.dbg.Emulator(), addr)
}

func (bc *baseContext[Impl]) LocalStore(key, val any) {
	bc.storage.Store(key, val)
}

func (bc *baseContext[Impl]) LocalLoad(key any) (any, bool) {
	return bc.storage.Load(key)
}

func (bc *baseContext[Impl]) LocalDelete(key any) {
	bc.storage.Delete(key)
}
