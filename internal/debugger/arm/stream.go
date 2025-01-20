package arm

import (
	"errors"
	"io"
	"math"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	emu_arm "github.com/wnxd/microdbg/emulator/arm"
	emu_arm64 "github.com/wnxd/microdbg/emulator/arm64"
	"github.com/wnxd/microdbg/encoding"
	internal "github.com/wnxd/microdbg/internal/debugger"
)

const ARG_REG_COUNT = 4

type regStream struct {
	dbg   debugger.Debugger
	ctx   debugger.RegisterContext
	stoff int
	groff int
	vroff int
	value uint64
	stack interface {
		io.ReaderAt
		io.WriterAt
	}
}

func (rs *regStream) Align() {
	rs.stoff = debugger.Align(rs.stoff, 4)
	rs.groff = debugger.Align(rs.groff, POINTER_SIZE)
}

func (rs *regStream) BlockSize() int {
	return POINTER_SIZE
}

func (rs *regStream) Offset() uint64 {
	return 0
}

func (rs *regStream) Skip(n int) error {
	if rs.groff < ARG_REG_COUNT*POINTER_SIZE {
		rs.groff += n
	} else {
		rs.stoff += n
	}
	return nil
}

func (rs *regStream) Read(b []byte) (int, error) {
	if rs.groff >= ARG_REG_COUNT*POINTER_SIZE {
		n, err := rs.stack.ReadAt(b, int64(rs.stoff))
		rs.stoff += n
		return n, err
	}
	var i int
	count := rs.groff / POINTER_SIZE
	if i = rs.groff % POINTER_SIZE; i > 0 {
		i = copy(b, internal.ToPtrRaw(&rs.value)[i:])
		rs.groff += i
		count++
	}
	for i < len(b) {
		if rs.groff >= ARG_REG_COUNT*POINTER_SIZE {
			n, err := rs.stack.ReadAt(b[i:], int64(rs.stoff))
			rs.stoff += n
			return i + n, err
		}
		var err error
		rs.value, err = rs.ctx.RegRead(emu_arm.ARM_REG_R0 + emulator.Reg(count))
		if err != nil {
			return i, err
		}
		n := copy(b[i:], internal.ToPtrRaw(&rs.value))
		i += n
		rs.groff += n
		count++
	}
	return i, nil
}

func (rs *regStream) ReadFloat() (float32, error) {
	if rs.vroff >= ARG_REG_COUNT {
		var f float32
		_, err := rs.stack.ReadAt(internal.ToPtrRaw(&f), int64(rs.stoff))
		rs.stoff += 4
		return f, err
	}
	value, err := rs.ctx.RegRead(emu_arm.ARM_REG_S0 + emulator.Reg(rs.vroff))
	if err != nil {
		return 0, err
	}
	rs.vroff++
	return math.Float32frombits(uint32(value)), nil
}

func (rs *regStream) ReadDouble() (float64, error) {
	if rs.vroff >= ARG_REG_COUNT {
		var d float64
		_, err := rs.stack.ReadAt(internal.ToPtrRaw(&d), int64(rs.stoff))
		rs.stoff += 8
		return d, err
	}
	value, err := rs.ctx.RegRead(emu_arm.ARM_REG_D0 + emulator.Reg(rs.vroff))
	if err != nil {
		return 0, err
	}
	rs.vroff++
	return math.Float64frombits(value), nil
}

func (rs *regStream) ReadString() (string, error) {
	return "", errors.ErrUnsupported
}

func (rs *regStream) ReadStream() (encoding.Stream, error) {
	var addr uint64
	_, err := rs.Read(internal.ToPtrRaw(&addr))
	if err != nil {
		return nil, err
	}
	return internal.PointerStream(rs.dbg.ToPointer(addr), rs.ctx.StackAlloc, POINTER_SIZE), nil
}

func (rs *regStream) Write(b []byte) (int, error) {
	if rs.groff >= ARG_REG_COUNT*POINTER_SIZE {
		n, err := rs.stack.WriteAt(b, int64(rs.stoff))
		rs.stoff += n
		return n, err
	}
	var i int
	count := rs.groff / POINTER_SIZE
	if i = rs.groff % POINTER_SIZE; i > 0 {
		i = copy(internal.ToPtrRaw(&rs.value)[i:], b)
		err := rs.ctx.RegWrite(emu_arm.ARM_REG_R0+emulator.Reg(count), rs.value)
		if err != nil {
			return 0, err
		}
		rs.groff += i
		count++
	}
	for i < len(b) {
		if rs.groff >= ARG_REG_COUNT*POINTER_SIZE {
			n, err := rs.stack.WriteAt(b[i:], int64(rs.stoff))
			rs.stoff += n
			return i + n, err
		}
		n := copy(internal.ToPtrRaw(&rs.value), b[i:])
		err := rs.ctx.RegWrite(emu_arm.ARM_REG_R0+emulator.Reg(count), rs.value)
		if err != nil {
			return i, err
		}
		i += n
		rs.groff += n
		count++
	}
	return i, nil
}

func (rs *regStream) WriteFloat(f float32) error {
	if rs.vroff >= ARG_REG_COUNT {
		_, err := rs.stack.WriteAt(internal.ToPtrRaw(&f), int64(rs.stoff))
		rs.stoff += 4
		return err
	}
	err := rs.ctx.RegWrite(emu_arm.ARM_REG_S0+emulator.Reg(rs.vroff), uint64(math.Float32bits(f)))
	if err != nil {
		return err
	}
	rs.vroff++
	return nil
}

func (rs *regStream) WriteDouble(d float64) error {
	if rs.vroff >= ARG_REG_COUNT {
		_, err := rs.stack.WriteAt(internal.ToPtrRaw(&d), int64(rs.stoff))
		rs.stoff += 8
		return err
	}
	err := rs.ctx.RegWrite(emu_arm64.ARM64_REG_S0+emulator.Reg(rs.vroff), math.Float64bits(d))
	if err != nil {
		return err
	}
	rs.vroff++
	return nil
}

func (rs *regStream) WriteString(string) error {
	return errors.ErrUnsupported
}

func (rs *regStream) WriteStream(size int) (encoding.Stream, error) {
	ptr, err := rs.ctx.StackAlloc(uint64(size))
	if err != nil {
		return nil, err
	}
	addr := ptr.Address()
	_, err = rs.Write(internal.ToPtrRaw(&addr))
	if err != nil {
		return nil, err
	}
	return internal.PointerStream(ptr, rs.ctx.StackAlloc, POINTER_SIZE), nil
}
