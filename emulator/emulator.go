package emulator

import (
	"io"
	"unsafe"
)

type Emulator interface {
	io.Closer
	Arch() Arch
	ByteOrder() ByteOrder
	PageSize() uint64
	MemMap(addr, size uint64, prot MemProt) error
	MemMapPtr(addr, size uint64, prot MemProt, ptr unsafe.Pointer) error
	MemUnmap(addr, size uint64) error
	MemProtect(addr, size uint64, prot MemProt) error
	MemRegions() ([]MemRegion, error)
	MemRead(addr, size uint64) ([]byte, error)
	MemWrite(addr uint64, data []byte) error
	MemReadPtr(addr, size uint64, ptr unsafe.Pointer) error
	MemWritePtr(addr, size uint64, ptr unsafe.Pointer) error
	RegisterContext
	Start(begin, until uint64) error
	Stop() error
	ContextAlloc() (Context, error)
	Hook(typ HookType, callback any, data any, begin, end uint64) (Hook, error)
}
