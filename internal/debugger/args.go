package debugger

type Args func(...any) error

func (va Args) Extract(args ...any) error {
	return va(args...)
}
