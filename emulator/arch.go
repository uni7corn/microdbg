package emulator

type Arch int

const (
	ARCH_UNKNOWN Arch = iota
	ARCH_ARM
	ARCH_ARM64
	ARCH_X86
	ARCH_X86_64
)
