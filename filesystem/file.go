package filesystem

import "io/fs"

type File interface {
	Close() error
	Stat() (fs.FileInfo, error)
}

type ReadFile interface {
	File
	Read(b []byte) (n int, err error)
}

type WriteFile interface {
	File
	Write(b []byte) (n int, err error)
}

type ControlFile interface {
	File
	Control(op int, arg any) error
}
