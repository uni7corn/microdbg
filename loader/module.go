package loader

import (
	"encoding/binary"
	"io"

	"github.com/wnxd/microdbg/emulator"
)

type Module interface {
	io.Closer
	Name() string
	Arch() emulator.Arch
	ByteOrder() binary.ByteOrder
	Regions() []Region
	EntryAddr() uint64
	InitAddrs() []uint64
	Libraries() []string
	Relocations() []Relocation
	FindSymbol(name string) (uint64, error)
}
