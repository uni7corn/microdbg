package debugger

import (
	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/encoding"
)

type pointerStream struct {
	ptr   emulator.Pointer
	alloc func(uint64) (emulator.Pointer, error)
	size  int
}

func PointerStream(ptr emulator.Pointer, alloc func(uint64) (emulator.Pointer, error), size int) encoding.Stream {
	return &pointerStream{ptr, alloc, size}
}

func (ps *pointerStream) BlockSize() int {
	return ps.size
}

func (ps *pointerStream) Offset() uint64 {
	return ps.ptr.Address()
}

func (ps *pointerStream) Skip(n int) error {
	ps.ptr = ps.ptr.Add(uint64(n))
	return nil
}

func (ps *pointerStream) Read(b []byte) (int, error) {
	n, err := ps.ptr.ReadAt(b, 0)
	if err == nil {
		ps.Skip(n)
	}
	return n, err
}

func (ps *pointerStream) ReadFloat() (float32, error) {
	var f float32
	_, err := ps.Read(ToPtrRaw(&f))
	return f, err
}

func (ps *pointerStream) ReadDouble() (float64, error) {
	var d float64
	_, err := ps.Read(ToPtrRaw(&d))
	return d, err
}

func (ps *pointerStream) ReadString() (string, error) {
	str, err := ps.ptr.MemReadString()
	if err == nil {
		ps.Skip(len(str) + 1)
	}
	return str, err
}

func (ps *pointerStream) ReadStream() (encoding.Stream, error) {
	ptr, err := ps.ptr.MemReadPointer()
	if err != nil {
		return nil, err
	}
	ps.Skip(ps.size)
	return PointerStream(ptr, ps.alloc, ps.size), nil
}

func (ps *pointerStream) Write(b []byte) (int, error) {
	n, err := ps.ptr.WriteAt(b, 0)
	if err == nil {
		ps.Skip(n)
	}
	return n, err
}

func (ps *pointerStream) WriteFloat(f float32) error {
	_, err := ps.Write(ToPtrRaw(&f))
	return err
}

func (ps *pointerStream) WriteDouble(d float64) error {
	_, err := ps.Write(ToPtrRaw(&d))
	return err
}

func (ps *pointerStream) WriteString(str string) error {
	_, err := ps.Write([]byte(str))
	if err != nil {
		return err
	}
	_, err = ps.Write([]byte{0})
	return err
}

func (ps *pointerStream) WriteStream(size int) (encoding.Stream, error) {
	ptr, err := ps.alloc(uint64(size))
	if err != nil {
		return nil, err
	}
	addr := ptr.Address()
	ps.Write(ToPtrRaw(&addr)[:ps.size])
	return PointerStream(ptr, ps.alloc, ps.size), nil
}
