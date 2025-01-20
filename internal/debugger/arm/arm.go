package arm

import (
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	emu_arm "github.com/wnxd/microdbg/emulator/arm"
	"github.com/wnxd/microdbg/encoding"
	internal "github.com/wnxd/microdbg/internal/debugger"
)

const (
	ARM_STACK_SIZE = 5 * 0x1000
	POINTER_SIZE   = 4
)

type ArmDbg struct {
	internal.Dbg
}

func NewArmDebugger(emu emulator.Emulator) (debugger.Debugger, error) {
	dbg := new(ArmDbg)
	err := dbg.Init(dbg, emu)
	if err != nil {
		return nil, err
	}
	return dbg, nil
}

func (dbg *ArmDbg) Init(impl internal.Debugger, emu emulator.Emulator) error {
	return dbg.Dbg.Init(impl, emu)
}

func (dbg *ArmDbg) Close() error {
	return dbg.Dbg.Close()
}

func (dbg *ArmDbg) PointerSize() uint64 {
	return POINTER_SIZE
}

func (dbg *ArmDbg) StackAlign() uint64 {
	return 8
}

func (dbg *ArmDbg) PC() emulator.Reg {
	return emu_arm.ARM_REG_PC
}

func (dbg *ArmDbg) SP() emulator.Reg {
	return emu_arm.ARM_REG_SP
}

func (dbg *ArmDbg) Args(ctx debugger.RegisterContext, calling debugger.Calling) (debugger.Args, error) {
	switch calling {
	case debugger.Calling_Default:
	case debugger.Calling_Fastcall:
	default:
		return nil, debugger.ErrCallingUnsupported
	}
	stackAddr, err := ctx.RegRead(emu_arm.ARM_REG_SP)
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

func (dbg *ArmDbg) ArgWrite(ctx debugger.RegisterContext, calling debugger.Calling, args ...any) error {
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

func (dbg *ArmDbg) RetExtract(ctx debugger.RegisterContext, val any) error {
	if internal.GetPtr(val) == nil {
		return debugger.ErrArgumentInvalid
	}
	stream := &regStream{dbg: dbg, ctx: ctx}
	return encoding.Decode(stream, val)
}

func (dbg *ArmDbg) RetWrite(ctx debugger.RegisterContext, val any) error {
	if internal.GetPtr(val) == nil {
		return ctx.RegWrite(emu_arm.ARM_REG_R0, 0)
	}
	stream := &regStream{dbg: dbg, ctx: ctx}
	return encoding.Encode(stream, val)
}

func (dbg *ArmDbg) Return(ctx debugger.RegisterContext) error {
	lr, err := ctx.RegRead(emu_arm.ARM_REG_LR)
	if err != nil {
		return err
	}
	return ctx.RegWrite(emu_arm.ARM_REG_PC, lr)
}

func (dbg *ArmDbg) InitStack() (uint64, error) {
	region, err := dbg.MapAlloc(ARM_STACK_SIZE, emulator.MEM_PROT_READ|emulator.MEM_PROT_WRITE)
	if err != nil {
		return 0, err
	}
	stack := region.Addr + ARM_STACK_SIZE
	return stack, nil
}

func (dbg *ArmDbg) CloseStack(stack uint64) error {
	begin := stack - ARM_STACK_SIZE
	return dbg.MapFree(begin, ARM_STACK_SIZE)
}

func (dbg *ArmDbg) TaskControl(task debugger.Task, addr uint64) (debugger.ControlHandler, error) {
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
	ctx.RegWrite(emu_arm.ARM_REG_PC, addr)
	ctx.RegWrite(emu_arm.ARM_REG_LR, ctrl.Addr())
	return ctrl, nil
}
