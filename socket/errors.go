package socket

import "errors"

var (
	ErrNotBind        = errors.New("socket not bind")
	ErrAlreadyBind    = errors.New("socket already bind")
	ErrNotListen      = errors.New("socket not listen")
	ErrAlreadyListen  = errors.New("socket already listen")
	ErrNotConnect     = errors.New("socket not connect")
	ErrAlreadyConnect = errors.New("socket already connect")
)
