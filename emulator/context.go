package emulator

import (
	"io"
	"unsafe"
)

type Context interface {
	io.Closer
	Save() error
	Restore() error
	RegisterContext
	Clone() (Context, error)
}

type RegisterContext interface {
	RegRead(reg Reg) (uint64, error)
	RegWrite(reg Reg, value uint64) error
	RegReadPtr(reg Reg, ptr unsafe.Pointer) error
	RegWritePtr(reg Reg, ptr unsafe.Pointer) error
	RegReadBatch(regs ...Reg) ([]uint64, error)
	RegWriteBatch(regs []Reg, vals []uint64) error
}
