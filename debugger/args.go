package debugger

type Args interface {
	Extract(...any) error
}
