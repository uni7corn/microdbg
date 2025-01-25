package debugger

import (
	"unsafe"

	"github.com/wnxd/microdbg/emulator"
)

type MemoryManager interface {
	MemMap(addr, size uint64, prot emulator.MemProt) (emulator.MemRegion, error)
	MemUnmap(addr uint64, size uint64) error
	MemProtect(addr, size uint64, prot emulator.MemProt) error
	MapAlloc(size uint64, prot emulator.MemProt) (emulator.MemRegion, error)
	MapFree(addr uint64, size uint64) error
	MemAlloc(size uint64) (uint64, error)
	MemFree(addr uint64) error
	MemSize(addr uint64) uint64
	ToPointer(addr uint64) emulator.Pointer
	MemImport(val any) ([]uint64, error)
	MemWrite(addr uint64, val any) ([]uint64, error)
	MemExtract(addr uint64, val any) error
	MemBind(p unsafe.Pointer, size uint64) (uint64, error)
	MemUnbind(addr uint64) error
}
