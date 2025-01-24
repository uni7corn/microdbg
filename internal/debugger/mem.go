package debugger

import (
	"maps"
	"slices"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/encoding"
)

type memBlock struct {
	addr uint64
	size uint64
	prev *memBlock
	next *memBlock
}

type memoryManager struct {
	mapAddr uint64
	mapMu   sync.Mutex
	maps    map[uint64]uint64
	memMu   sync.Mutex
	used    map[uint64]uint64
	free    *memBlock
	bindMu  sync.Mutex
	binds   map[uint64]debugger.HookHandler
}

var blockPool = sync.Pool{
	New: func() any {
		return new(memBlock)
	},
}

func (mm *memoryManager) ctor() {
	mm.mapAddr = 0x400000
	mm.maps = make(map[uint64]uint64)
	mm.used = make(map[uint64]uint64)
	mm.binds = make(map[uint64]debugger.HookHandler)
}

func (mm *memoryManager) dtor(dbg Debugger) {
	emu := dbg.Emulator()
	mm.bindMu.Lock()
	for _, b := range mm.binds {
		b.Close()
	}
	clear(mm.binds)
	mm.bindMu.Unlock()
	mm.mapMu.Lock()
	for addr, size := range mm.maps {
		emu.MemUnmap(addr, size)
	}
	clear(mm.maps)
	mm.mapMu.Unlock()
}

func (mm *memoryManager) mapStore(addr, size uint64) {
	mm.mapMu.Lock()
	mm.maps[addr] = size
	mm.mapMu.Unlock()
}

func (mm *memoryManager) mapDelete(addr, size uint64) {
	mm.mapMu.Lock()
	defer mm.mapMu.Unlock()
	for {
		if valid, ok := mm.maps[addr]; ok {
			delete(mm.maps, addr)
			if size < valid {
				mm.maps[addr+size] = valid - size
			} else if size > valid {
				addr += valid
				size -= valid
				continue
			}
			return
		}
		break
	}
	end := addr + size
	for _, start := range slices.Sorted(maps.Keys(mm.maps)) {
		blockEnd := start + mm.maps[start]
		overlapStart, overlapEnd, ok := calcOverlap(start, blockEnd, addr, end)
		if !ok {
			continue
		}
		if overlapStart > start {
			mm.maps[start] = overlapStart - start
		} else {
			delete(mm.maps, start)
		}
		if overlapEnd < blockEnd {
			mm.maps[overlapEnd] = blockEnd - overlapEnd
		} else if overlapEnd < end {
			addr = overlapEnd
			continue
		}
		break
	}
}

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

func (mm *memoryManager) memAlloc(dbg Debugger, size uint64) (uint64, error) {
	if size == 0 {
		return 0, debugger.ErrArgumentInvalid
	}
	mm.memMu.Lock()
	defer mm.memMu.Unlock()
	for b := range mm.free.Range {
		if b.size >= size {
			addr := b.addr
			b.size -= size
			if b.size == 0 {
				if mm.free == b {
					mm.free = b.next
				}
				b.Remove()
			} else {
				b.addr = addr + size
			}
			mm.used[addr] = size
			return addr, nil
		}
	}
	region, err := mm.mapAlloc(dbg, size, emulator.MEM_PROT_READ|emulator.MEM_PROT_WRITE)
	if err != nil {
		return 0, err
	} else if region.Size > size {
		mm.free = mm.free.InsertAfter(region.Addr+size, region.Size-size)
	}
	mm.used[region.Addr] = size
	return region.Addr, nil
}

func (mm *memoryManager) memFree(dbg Debugger, addr uint64) error {
	if addr == 0 {
		return debugger.ErrAddressInvalid
	}
	mm.memMu.Lock()
	defer mm.memMu.Unlock()
	size, ok := mm.used[addr]
	if !ok {
		return debugger.ErrAddressInvalid
	}
	delete(mm.used, addr)
	end := addr + size
	var b *memBlock
	for b = range mm.free.Range {
		if b.end() == addr {
			if b.prev == nil || b.prev.addr != end {
				b.size += size
			} else {
				b.prev.addr = b.addr
				b.prev.size += b.size + size
				b.Remove()
			}
			return nil
		} else if b.addr < end {
			nb := b.InsertAfter(addr, size)
			if mm.free == b {
				mm.free = nb
			}
			return nil
		}
	}
	if b.addr == end {
		b.addr = addr
		b.size += size
	} else {
		b.InsertBefore(addr, size)
	}
	return nil
}

func (mm *memoryManager) memSize(dbg Debugger, addr uint64) uint64 {
	mm.memMu.Lock()
	defer mm.memMu.Unlock()
	return mm.used[addr]
}

func (mm *memoryManager) memImport(dbg Debugger, val any) ([]uint64, error) {
	addr, err := mm.memAlloc(dbg, uint64(encoding.EncodeSize(int(dbg.PointerSize()), val)))
	if err != nil {
		return nil, err
	}
	addrs, err := mm.memWrite(dbg, addr, val)
	if err != nil {
		mm.memFree(dbg, addr)
		return nil, err
	}
	return append([]uint64{addr}, addrs...), nil
}

func (mm *memoryManager) memWrite(dbg Debugger, addr uint64, val any) ([]uint64, error) {
	var addrs []uint64
	stream := PointerStream(dbg.ToPointer(addr), func(size uint64) (emulator.Pointer, error) {
		addr, err := mm.memAlloc(dbg, size)
		if err != nil {
			return emulator.Pointer{}, err
		}
		addrs = append(addrs, addr)
		return dbg.ToPointer(addr), nil
	}, int(dbg.PointerSize()))
	err := encoding.Encode(stream, val)
	if err != nil {
		for _, addr := range addrs {
			mm.memFree(dbg, addr)
		}
		return nil, err
	}
	return addrs, nil
}

func (mm *memoryManager) memExtract(dbg Debugger, addr uint64, val any) error {
	stream := PointerStream(dbg.ToPointer(addr), nil, int(dbg.PointerSize()))
	return encoding.Decode(stream, val)
}

func (mm *memoryManager) memBind(dbg Debugger, p unsafe.Pointer, size uint64) (uint64, error) {
	addr, err := mm.memAlloc(dbg, size)
	if err != nil {
		return 0, err
	}
	hook, err := dbg.AddHook(emulator.HOOK_TYPE_MEM_READ|emulator.HOOK_TYPE_MEM_WRITE, func(ctx debugger.Context, typ emulator.HookType, addr, size, value uint64, data any) debugger.HookResult {
		offset := addr - data.(uint64)
		switch typ {
		case emulator.HOOK_TYPE_MEM_READ:
			ctx.ToPointer(addr).MemWritePtr(size, unsafe.Add(p, offset))
		case emulator.HOOK_TYPE_MEM_WRITE:
			copy(unsafe.Slice((*byte)(p), size)[offset:], ToPtrRaw(&value)[:size])
		}
		return debugger.HookResult_Next
	}, addr, addr, addr+size)
	if err != nil {
		mm.memFree(dbg, addr)
		return 0, err
	}
	mm.bindMu.Lock()
	mm.binds[addr] = hook
	mm.bindMu.Unlock()
	return addr, nil
}

func (mm *memoryManager) memUnbind(dbg Debugger, addr uint64) error {
	mm.bindMu.Lock()
	hook, ok := mm.binds[addr]
	if !ok {
		mm.bindMu.Unlock()
		return debugger.ErrAddressInvalid
	}
	delete(mm.binds, addr)
	mm.bindMu.Unlock()
	hook.Close()
	mm.memFree(dbg, addr)
	return nil
}

func (mb *memBlock) Range(yield func(*memBlock) bool) {
	for b := mb; b != nil; b = b.next {
		if !yield(b) {
			break
		}
	}
}

func (mb *memBlock) InsertAfter(addr, size uint64) *memBlock {
	b := blockPool.Get().(*memBlock)
	b.addr = addr
	b.size = size
	b.next = mb
	if mb != nil {
		b.prev = mb.prev
		if mb.prev != nil {
			mb.prev.next = b
		}
		mb.prev = b
	}
	return b
}

func (mb *memBlock) InsertBefore(addr, size uint64) *memBlock {
	b := blockPool.Get().(*memBlock)
	b.addr = addr
	b.size = size
	b.prev = mb
	if mb != nil {
		b.next = mb.next
		if mb.next != nil {
			mb.next.prev = b
		}
		mb.next = b
	}
	return b
}

func (mb *memBlock) Remove() {
	if mb.prev != nil {
		mb.prev.next = mb.next
	}
	if mb.next != nil {
		mb.next.prev = mb.prev
	}
	mb.prev = nil
	mb.next = nil
	blockPool.Put(mb)
}

func (mb *memBlock) end() uint64 {
	return mb.addr + mb.size
}

func (dbg *Dbg) MemMap(addr, size uint64, prot emulator.MemProt) (emulator.MemRegion, error) {
	return dbg.memoryManager.memMap(dbg.impl, addr, size, prot)
}

func (dbg *Dbg) MemUnmap(addr, size uint64) error {
	return dbg.memoryManager.memUnmap(dbg.impl, addr, size)
}

func (dbg *Dbg) MemProtect(addr, size uint64, prot emulator.MemProt) error {
	return dbg.memoryManager.memProtect(dbg.impl, addr, size, prot)
}

func (dbg *Dbg) MapAlloc(size uint64, prot emulator.MemProt) (emulator.MemRegion, error) {
	return dbg.memoryManager.mapAlloc(dbg.impl, size, prot)
}

func (dbg *Dbg) MapFree(addr, size uint64) error {
	return dbg.memoryManager.mapFree(dbg.impl, addr, size)
}

func (dbg *Dbg) MemAlloc(size uint64) (uint64, error) {
	return dbg.memoryManager.memAlloc(dbg.impl, size)
}

func (dbg *Dbg) MemFree(addr uint64) error {
	return dbg.memoryManager.memFree(dbg.impl, addr)
}

func (dbg *Dbg) MemSize(addr uint64) uint64 {
	return dbg.memoryManager.memSize(dbg.impl, addr)
}

func (dbg *Dbg) ToPointer(addr uint64) emulator.Pointer {
	return emulator.ToPointer(dbg.Emulator(), addr)
}

func (dbg *Dbg) MemImport(val any) ([]uint64, error) {
	return dbg.memoryManager.memImport(dbg.impl, val)
}

func (dbg *Dbg) MemWrite(addr uint64, val any) ([]uint64, error) {
	return dbg.memoryManager.memWrite(dbg.impl, addr, val)
}

func (dbg *Dbg) MemExtract(addr uint64, val any) error {
	return dbg.memoryManager.memExtract(dbg.impl, addr, val)
}

func (dbg *Dbg) MemBind(p unsafe.Pointer, size uint64) (uint64, error) {
	return dbg.memoryManager.memBind(dbg.impl, p, size)
}

func (dbg *Dbg) MemUnbind(addr uint64) error {
	return dbg.memoryManager.memUnbind(dbg.impl, addr)
}

func calcOverlap(min1, max1, min2, max2 uint64) (uint64, uint64, bool) {
	if max1 < min2 || max2 < min1 {
		return 0, 0, false
	}
	overlapMin := max(min1, min2)
	overlapMax := min(max1, max2)
	return overlapMin, overlapMax, true
}
