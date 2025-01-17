package debugger

import (
	"github.com/wnxd/microdbg/emulator"
)

type DbgCtor func(emulator.Emulator) (Debugger, error)

var dbgMap = make(map[emulator.Arch]DbgCtor)

func Register(arch emulator.Arch, ctor DbgCtor) bool {
	if _, ok := dbgMap[arch]; ok {
		return false
	}
	dbgMap[arch] = ctor
	return true
}
