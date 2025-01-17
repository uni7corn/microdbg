package arm64

import (
	"reflect"

	"github.com/wnxd/microdbg/emulator"
	"github.com/wnxd/microdbg/internal/debugger/extend"
)

func NewExtendDebugger[D extend.ExtendDebugger](emu emulator.Emulator) (D, error) {
	impl := reflect.New(reflect.TypeFor[D]().Elem()).Interface().(D)
	return impl, impl.ExtendInit(emu)
}
