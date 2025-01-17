package debugger

import (
	"context"
	"io"
)

type Symbol struct {
	Name  string
	Value uint64
}

type SymbolIter interface {
	Symbols(yield func(Symbol) bool)
}

type Module interface {
	io.Closer
	Name() string
	Region() (uint64, uint64)
	BaseAddr() uint64
	EntryAddr() uint64
	Init(ctx context.Context) error
	FindSymbol(name string) (uint64, error)
}

type ModuleManager interface {
	Load(module Module)
	Unload(module Module)
	FindModule(name string) (Module, error)
	FindModuleByAddr(addr uint64) (Module, error)
	FindSymbol(name string) (Module, uint64, error)
	GetModule(addr uint64) Module
}

var InternalModule Module = new(module)

type module struct{}

func (module) Close() error                           { return nil }
func (module) Name() string                           { return "" }
func (module) Region() (uint64, uint64)               { return 0, 0 }
func (module) BaseAddr() uint64                       { return 0 }
func (module) EntryAddr() uint64                      { return 0 }
func (module) Init(ctx context.Context) error         { return nil }
func (module) FindSymbol(name string) (uint64, error) { return 0, ErrSymbolNotFound }
