package emulator

type ByteOrder int

const (
	BO_LITTLE_ENDIAN ByteOrder = iota
	BO_BIG_ENDIAN
)

type MemProt int

const (
	MEM_PROT_NONE MemProt = 0
	MEM_PROT_READ MemProt = 1 << (iota - 1)
	MEM_PROT_WRITE
	MEM_PROT_EXEC

	MEM_PROT_ALL = MEM_PROT_READ | MEM_PROT_WRITE | MEM_PROT_EXEC
)

type MemRegion struct {
	Addr, Size uint64
	Prot       MemProt
}
