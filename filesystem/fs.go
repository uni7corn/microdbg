package filesystem

import (
	"io/fs"
	"os"
)

type FileFlag int

const (
	O_RDONLY = FileFlag(os.O_RDONLY)
	O_WRONLY = FileFlag(os.O_WRONLY)
	O_RDWR   = FileFlag(os.O_RDWR)
	O_APPEND = FileFlag(os.O_APPEND)
	O_CREATE = FileFlag(os.O_CREATE)
	O_EXCL   = FileFlag(os.O_EXCL)
	O_SYNC   = FileFlag(os.O_SYNC)
	O_TRUNC  = FileFlag(os.O_TRUNC)
)

type FS interface {
	fs.FS
	OpenFile(name string, flag FileFlag, perm fs.FileMode) (File, error)
}

type DirFS interface {
	FS
	ReadDir(name string) ([]fs.DirEntry, error)
	Mkdir(name string, perm fs.FileMode) (DirFS, error)
}

type ReadlinkFS interface {
	DirFS
	Readlink(name string) (string, error)
}

func Open(f FS, name string) (fs.File, error) {
	file, err := f.OpenFile(name, O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return file.(fs.File), nil
}
