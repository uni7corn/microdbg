package extend

import (
	"errors"
	"reflect"

	"github.com/wnxd/microdbg/emulator"
	arm "github.com/wnxd/microdbg/internal/debugger/arm"
	arm64 "github.com/wnxd/microdbg/internal/debugger/arm64"
	"github.com/wnxd/microdbg/internal/debugger/extend"
)

type ExtendDebugger extend.ExtendDebugger

var extendType = reflect.TypeFor[ExtendDebugger]()

func New[D ExtendDebugger](emu emulator.Emulator) (dbg D, err error) {
	typ := reflect.TypeOf(dbg)
	var isPtr bool
	switch typ.Kind() {
	case reflect.Pointer:
		dbg = reflect.New(reflect.TypeFor[D]().Elem()).Interface().(D)
		typ = typ.Elem()
		isPtr = true
	case reflect.Struct:
	default:
		return dbg, errors.ErrUnsupported
	}
	if field, ok := typ.FieldByName("ExtendDebugger"); ok && field.Type == extendType {
		var d ExtendDebugger
		d, err = newDbg[D](emu.Arch())
		if err != nil {
			return
		}
		v := reflect.ValueOf(dbg)
		if isPtr {
			v = v.Elem()
		}
		v.FieldByIndex(field.Index).Set(reflect.ValueOf(d))
	}
	err = dbg.Init(dbg, emu)
	return
}

func newDbg[D ExtendDebugger](arch emulator.Arch) (ExtendDebugger, error) {
	switch arch {
	case emulator.ARCH_ARM:
		return new(arm.ArmDbg), nil
	case emulator.ARCH_ARM64:
		return new(arm64.Arm64Dbg), nil
	case emulator.ARCH_X86:
	case emulator.ARCH_X86_64:
	}
	return nil, emulator.ErrArchUnsupported
}
