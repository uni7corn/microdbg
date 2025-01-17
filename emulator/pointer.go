package emulator

import (
	"slices"
	"unsafe"
)

type Uintptr32 = uint32
type Uintptr64 = uint64

type Pointer struct {
	emu  Emulator
	addr uint64
}

func ToPointer(emu Emulator, addr uint64) Pointer {
	return Pointer{emu, addr}
}

func (p Pointer) IsNil() bool {
	return p.addr == 0
}

func (p Pointer) Address() uint64 {
	return p.addr
}

func (p Pointer) Add(offset uint64) Pointer {
	return Pointer{p.emu, p.addr + offset}
}

func (p Pointer) Sub(offset uint64) Pointer {
	return Pointer{p.emu, p.addr - offset}
}

func (p Pointer) MemRead(size uint64) ([]byte, error) {
	return p.emu.MemRead(p.addr, size)
}

func (p Pointer) MemWrite(data []byte) error {
	return p.emu.MemWrite(p.addr, data)
}

func (p Pointer) MemReadPtr(size uint64, ptr unsafe.Pointer) error {
	return p.emu.MemReadPtr(p.addr, size, ptr)
}

func (p Pointer) MemWritePtr(size uint64, ptr unsafe.Pointer) error {
	return p.emu.MemWritePtr(p.addr, size, ptr)
}

func (p Pointer) MemReadString() (string, error) {
	var data []byte
	var buf [0x10]byte
	size := uint64(len(buf))
	for begin := p.addr; ; begin += size {
		err := p.emu.MemReadPtr(begin, size, unsafe.Pointer(unsafe.SliceData(buf[:])))
		if err != nil {
			return "", err
		}
		i := slices.Index(buf[:], 0)
		if i == -1 {
			data = append(data, buf[:]...)
		} else {
			data = append(data, buf[:i]...)
			break
		}
	}
	return string(data), nil
}

func (p Pointer) MemReadPointer() (ptr Pointer, err error) {
	var size uint64
	switch p.emu.Arch() {
	case ARCH_ARM, ARCH_X86:
		size = 4
	case ARCH_ARM64, ARCH_X86_64:
		size = 8
	default:
		err = ErrArchUnsupported
		return
	}
	var addr uint64
	err = p.MemReadPtr(size, unsafe.Pointer(&addr))
	if err != nil {
		return
	}
	ptr.emu, ptr.addr = p.emu, addr
	return
}

func (p Pointer) ReadAt(b []byte, off int64) (n int, err error) {
	return len(b), p.emu.MemReadPtr(p.addr+uint64(off), uint64(len(b)), unsafe.Pointer(unsafe.SliceData(b)))
}

func (p Pointer) WriteAt(b []byte, off int64) (n int, err error) {
	return len(b), p.emu.MemWritePtr(p.addr+uint64(off), uint64(len(b)), unsafe.Pointer(unsafe.SliceData(b)))
}
