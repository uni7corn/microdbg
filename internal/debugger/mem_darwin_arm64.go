//go:build darwin && arm64

package debugger

import (
	"sync/atomic"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
)

func (mm *memoryManager) memMap(dbg Debugger, addr, size uint64, prot emulator.MemProt) (emulator.MemRegion, error) {
	emu := dbg.Emulator()
	addr = debugger.Align(addr, emu.PageSize())
	size = debugger.Align(size, emu.PageSize())
	var err error
	dbg.mainThreadRun(func() {
		err = emu.MemMap(addr, size, prot)
	})
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
	var err error
	dbg.mainThreadRun(func() {
		err = emu.MemUnmap(addr, size)
	})
	if err == nil {
		mm.mapDelete(addr, size)
	}
	return err
}

func (mm *memoryManager) memProtect(dbg Debugger, addr, size uint64, prot emulator.MemProt) error {
	emu := dbg.Emulator()
	size = debugger.Align(size, emu.PageSize())
	var err error
	dbg.mainThreadRun(func() {
		err = emu.MemProtect(addr, size, prot)
	})
	return err
}

func (mm *memoryManager) mapAlloc(dbg Debugger, size uint64, prot emulator.MemProt) (emulator.MemRegion, error) {
	emu := dbg.Emulator()
	size = debugger.Align(size, emu.PageSize())
	addr := atomic.AddUint64(&mm.mapAddr, size) - size
	var err error
	dbg.mainThreadRun(func() {
		err = emu.MemMap(addr, size, prot)
	})
	if err != nil {
		return emulator.MemRegion{}, err
	}
	mm.mapStore(addr, size)
	return emulator.MemRegion{Addr: addr, Size: size, Prot: prot}, nil
}

func (mm *memoryManager) mapFree(dbg Debugger, addr, size uint64) error {
	emu := dbg.Emulator()
	size = debugger.Align(size, emu.PageSize())
	var err error
	dbg.mainThreadRun(func() {
		err = emu.MemUnmap(addr, size)
	})
	if err == nil {
		mm.mapDelete(addr, size)
	}
	return err
}
