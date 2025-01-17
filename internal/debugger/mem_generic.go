//go:build !(darwin && arm64)

package debugger

import (
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
)

func (mm *memoryManager) memMap(dbg Debugger, addr, size uint64, prot emulator.MemProt) (emulator.MemRegion, error) {
	emu := dbg.Emulator()
	addr = debugger.Align(addr, emu.PageSize())
	size = debugger.Align(size, emu.PageSize())
	err := emu.MemMap(addr, size, prot)
	if err != nil {
		return emulator.MemRegion{}, err
	}
	mm.mapStore(addr, size)
	return emulator.MemRegion{Addr: addr, Size: size, Prot: prot}, nil
}

func (mm *memoryManager) memUnmap(dbg Debugger, addr, size uint64) error {
	emu := dbg.Emulator()
	addr = debugger.Align(addr, emu.PageSize())
	size = debugger.Align(size, emu.PageSize())
	err := emu.MemUnmap(addr, size)
	if err == nil {
		mm.mapDelete(addr, size)
	}
	return err
}

func (mm *memoryManager) memProtect(dbg Debugger, addr, size uint64, prot emulator.MemProt) error {
	emu := dbg.Emulator()
	size = debugger.Align(size, emu.PageSize())
	return emu.MemProtect(addr, size, prot)
}

func (mm *memoryManager) mapAlloc(dbg Debugger, size uint64, prot emulator.MemProt) (emulator.MemRegion, error) {
	emu := dbg.Emulator()
	size = debugger.Align(size, emu.PageSize())
	addr := mm.mapAddr
	mm.mapAddr += size
	err := emu.MemMap(addr, size, prot)
	if err != nil {
		return emulator.MemRegion{}, err
	}
	mm.mapStore(addr, size)
	return emulator.MemRegion{Addr: addr, Size: size, Prot: prot}, nil
}

func (mm *memoryManager) mapFree(dbg Debugger, addr, size uint64) error {
	emu := dbg.Emulator()
	size = debugger.Align(size, emu.PageSize())
	err := emu.MemUnmap(addr, size)
	if err == nil {
		mm.mapDelete(addr, size)
	}
	return err
}
