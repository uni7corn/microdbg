package arm

import (
	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/emulator"
	emu_arm "github.com/wnxd/microdbg/emulator/arm"
	internal "github.com/wnxd/microdbg/internal/debugger"
)

const ARM_STACK_SIZE = 5 * 0x1000

type armDbg struct {
	internal.Dbg[*armDbg]
}

func NewArmDebugger(emu emulator.Emulator) (debugger.Debugger, error) {
	dbg := new(armDbg)
	err := dbg.Init(emu)
	if err != nil {
		return nil, err
	}
	return dbg, nil
}

func (dbg *armDbg) PointerSize() uint64 {
	return 4
}

func (dbg *armDbg) StackAlign() uint64 {
	return 8
}

func (dbg *armDbg) PC() emulator.Reg {
	return emu_arm.ARM_REG_PC
}

func (dbg *armDbg) SP() emulator.Reg {
	return emu_arm.ARM_REG_SP
}

func (dbg *armDbg) Args(debugger.RegisterContext, debugger.Calling) (debugger.Args, error) {
	return nil, debugger.ErrArgumentInvalid
}

func (dbg *armDbg) ArgWrite(debugger.RegisterContext, debugger.Calling, ...any) error {
	return debugger.ErrArgumentInvalid
}

func (dbg *armDbg) RetExtract(debugger.RegisterContext, any) error {
	return debugger.ErrArgumentInvalid
}

func (dbg *armDbg) RetWrite(debugger.RegisterContext, any) error {
	return debugger.ErrArgumentInvalid
}

func (dbg *armDbg) Return(ctx debugger.RegisterContext) error {
	lr, err := ctx.RegRead(emu_arm.ARM_REG_LR)
	if err != nil {
		return err
	}
	return ctx.RegWrite(emu_arm.ARM_REG_PC, lr)
}

func (dbg *armDbg) InitStack() (uint64, error) {
	region, err := dbg.MapAlloc(ARM_STACK_SIZE, emulator.MEM_PROT_READ|emulator.MEM_PROT_WRITE)
	if err != nil {
		return 0, err
	}
	stack := region.Addr + ARM_STACK_SIZE
	return stack, nil
}

func (dbg *armDbg) CloseStack(stack uint64) error {
	begin := stack - ARM_STACK_SIZE
	return dbg.MapFree(begin, ARM_STACK_SIZE)
}

func (dbg *armDbg) TaskControl(task debugger.Task, addr uint64) (debugger.ControlHandler, error) {
	return nil, debugger.ErrCallingUnsupported
}
