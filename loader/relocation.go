package loader

type Relocation interface {
	rel()
}

type RelocationValue struct {
	Addr, Size, Value uint64
}

type RelocationImport struct {
	Addr, Size      uint64
	Symbol, Library string
}

func (*RelocationValue) rel()  {}
func (*RelocationImport) rel() {}
