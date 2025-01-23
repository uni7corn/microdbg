package arm64

import (
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	emu_arm64 "github.com/wnxd/microdbg/emulator/arm64"
	"github.com/wnxd/microdbg/encoding"
	internal "github.com/wnxd/microdbg/internal/debugger"
	inter_arm "github.com/wnxd/microdbg/internal/debugger/arm"
)

const (
	ARM64_STACK_SIZE = inter_arm.ARM_STACK_SIZE * 2
	POINTER_SIZE     = 8
)

type Arm64Dbg struct {
	internal.Dbg
}

func NewArm64Debugger(emu emulator.Emulator) (debugger.Debugger, error) {
	dbg := new(Arm64Dbg)
	err := dbg.Init(dbg, emu)
	if err != nil {
		return nil, err
	}
	return dbg, nil
}

func (dbg *Arm64Dbg) Init(impl internal.Debugger, emu emulator.Emulator) error {
	err := dbg.Dbg.Init(impl, emu)
	if err != nil {
		return err
	}
	dbg.enableVFP()
	return nil
}

func (dbg *Arm64Dbg) Close() error {
	return dbg.Dbg.Close()
}

func (dbg *Arm64Dbg) PointerSize() uint64 {
	return POINTER_SIZE
}

func (dbg *Arm64Dbg) StackAlign() uint64 {
	return 16
}

func (dbg *Arm64Dbg) PC() emulator.Reg {
	return emu_arm64.ARM64_REG_PC
}

func (dbg *Arm64Dbg) SP() emulator.Reg {
	return emu_arm64.ARM64_REG_SP
}

func (dbg *Arm64Dbg) Args(ctx debugger.RegisterContext, calling debugger.Calling) (debugger.Args, error) {
	switch calling {
	case debugger.Calling_Default:
	case debugger.Calling_Fastcall:
	default:
		return nil, debugger.ErrCallingUnsupported
	}
	stackAddr, err := ctx.RegRead(emu_arm64.ARM64_REG_SP)
	if err != nil {
		return nil, err
	}
	var index int
	stream := &regStream{dbg: dbg, ctx: ctx, stack: dbg.ToPointer(stackAddr)}
	return internal.Args(func(args ...any) error {
		for _, arg := range args {
			err := encoding.Decode(stream, arg)
			if err != nil {
				return err
			}
			stream.Align()
			index++
		}
		return nil
	}), nil
}

func (dbg *Arm64Dbg) ArgWrite(ctx debugger.RegisterContext, calling debugger.Calling, args ...any) error {
	switch calling {
	case debugger.Calling_Default:
	case debugger.Calling_Fastcall:
	default:
		return debugger.ErrCallingUnsupported
	}
	var buf internal.Buffer
	stream := &regStream{dbg: dbg, ctx: ctx, stack: &buf}
	for _, arg := range args {
		err := encoding.Encode(stream, arg)
		if err != nil {
			return err
		}
		stream.Align()
	}
	if stream.stoff == 0 {
		return nil
	}
	ptr, err := ctx.StackAlloc(uint64(stream.stoff))
	if err != nil {
		return err
	}
	return ptr.MemWrite(buf)
}

func (dbg *Arm64Dbg) RetExtract(ctx debugger.RegisterContext, val any) error {
	if internal.GetPtr(val) == nil {
		return debugger.ErrArgumentInvalid
	}
	stream := &regStream{dbg: dbg, ctx: ctx}
	return encoding.Decode(stream, val)
}

func (dbg *Arm64Dbg) RetWrite(ctx debugger.RegisterContext, val any) error {
	if internal.GetPtr(val) == nil {
		return ctx.RegWrite(emu_arm64.ARM64_REG_X0, 0)
	}
	stream := &regStream{dbg: dbg, ctx: ctx}
	return encoding.Encode(stream, val)
}

func (dbg *Arm64Dbg) Return(ctx debugger.RegisterContext) error {
	lr, err := ctx.RegRead(emu_arm64.ARM64_REG_LR)
	if err != nil {
		return err
	}
	return ctx.RegWrite(emu_arm64.ARM64_REG_PC, lr)
}

func (dbg *Arm64Dbg) InitStack() (uint64, error) {
	region, err := dbg.MapAlloc(ARM64_STACK_SIZE, emulator.MEM_PROT_READ|emulator.MEM_PROT_WRITE)
	if err != nil {
		return 0, err
	}
	stack := region.Addr + ARM64_STACK_SIZE
	return stack, nil
}

func (dbg *Arm64Dbg) CloseStack(stack uint64) error {
	begin := stack - ARM64_STACK_SIZE
	return dbg.MapFree(begin, ARM64_STACK_SIZE)
}

func (dbg *Arm64Dbg) TaskControl(task debugger.Task, addr uint64) (debugger.ControlHandler, error) {
	ctrl, err := dbg.AddControl(func(ctx debugger.Context, data any) {
		task := data.(debugger.Task)
		if task.Context() != ctx {
			panic("call exception return")
		}
		task.CancelCause(debugger.TaskStatus_Done)
	}, task)
	if err != nil {
		return nil, err
	}
	ctx := task.Context()
	ctx.RegWrite(emu_arm64.ARM64_REG_PC, addr)
	ctx.RegWrite(emu_arm64.ARM64_REG_LR, ctrl.Addr())
	return ctrl, nil
}

func (dbg *Arm64Dbg) enableVFP() {
	emu := dbg.Emulator()
	val, _ := emu.RegRead(emu_arm64.ARM64_REG_CPACR_EL1)
	val |= 0x300000
	emu.RegWrite(emu_arm64.ARM64_REG_CPACR_EL1, val)
}
