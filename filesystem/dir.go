package filesystem

import "io/fs"

type DirFile interface {
	File
	ReadDir(n int) ([]fs.DirEntry, error)
}

type Dir interface {
	DirFile
	OpenFile(name string, flag FileFlag, perm fs.FileMode) (File, error)
	Mkdir(name string, perm fs.FileMode) error
}

type ReadlinkDir interface {
	DirFile
	Readlink(name string) (string, error)
}
