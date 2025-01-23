package loader

import (
	"io"

	"github.com/wnxd/microdbg/emulator"
)

type Region struct {
	Addr, Size    uint64
	Length, Align uint64
	Prot          emulator.MemProt
	io.ReaderAt
}
