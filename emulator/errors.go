package emulator

import "errors"

var (
	ErrArchUnsupported = errors.New("architecture unsupported")
	ErrArchMismatch    = errors.New("architecture mismatch")
)
