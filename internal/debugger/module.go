package debugger

import (
	"slices"
	"sync"

	"github.com/wnxd/microdbg/debugger"
)

type moduleManager struct {
	mu     sync.Mutex
	loaded []debugger.Module
}

func (mm *moduleManager) ctor() {
}

func (mm *moduleManager) dtor() {
	for _, module := range mm.loaded {
		module.Close()
	}
}

func (mm *moduleManager) Load(module debugger.Module) {
	mm.mu.Lock()
	if !slices.Contains(mm.loaded, module) {
		mm.loaded = append(mm.loaded, module)
	}
	mm.mu.Unlock()
}

func (mm *moduleManager) Unload(module debugger.Module) {
	mm.mu.Lock()
	mm.loaded = slices.DeleteFunc(mm.loaded, func(m debugger.Module) bool { return m == module })
	mm.mu.Unlock()
}

func (mm *moduleManager) FindModule(name string) (debugger.Module, error) {
	for _, module := range mm.loaded {
		if module.Name() == name {
			return module, nil
		}
	}
	return nil, debugger.ErrModuleNotFound
}

func (mm *moduleManager) FindModuleByAddr(addr uint64) (debugger.Module, error) {
	for _, module := range mm.loaded {
		begin, size := module.Region()
		if addr >= begin && addr < begin+size {
			return module, nil
		}
	}
	return nil, debugger.ErrModuleNotFound
}

func (mm *moduleManager) FindSymbol(name string) (debugger.Module, uint64, error) {
	for _, module := range mm.loaded {
		addr, err := module.FindSymbol(name)
		if err == nil {
			return module, addr, err
		}
	}
	return nil, 0, debugger.ErrSymbolNotFound
}

func (mm *moduleManager) GetModule(addr uint64) debugger.Module {
	for _, module := range mm.loaded {
		if module.BaseAddr() == addr {
			return module
		}
	}
	return nil
}
