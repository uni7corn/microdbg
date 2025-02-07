package debugger

import (
	"io/fs"

	"github.com/wnxd/microdbg/filesystem"
	"github.com/wnxd/microdbg/socket"
)

type FileHandler interface {
	OpenFile(name string, flag filesystem.FileFlag, perm fs.FileMode) (filesystem.File, error)
	Stat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	Mkdir(name string, perm fs.FileMode) (filesystem.DirFS, error)
	Readlink(name string) (string, error)
	NewSocket(network socket.Network) (socket.Socket, error)
}

type FileManager interface {
	AddFileHandler(handler FileHandler)
	RemoveFileHandler(handler FileHandler)
	CreateFileDescriptor(file filesystem.File) int
	CloseFileDescriptor(fd int) (filesystem.File, error)
	GetFile(fd int) (filesystem.File, error)
	DupFile(fd int) (int, error)
	Dup2File(oldfd, newfd int) error
	GetFS() filesystem.FS
	OpenFile(name string, flag filesystem.FileFlag, perm fs.FileMode) (filesystem.File, error)
	Stat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	Mkdir(name string, perm fs.FileMode) (filesystem.DirFS, error)
	Readlink(name string) (string, error)
	NewSocket(network socket.Network) (socket.Socket, error)
}

type DefaultFileHandler struct {
}

func (DefaultFileHandler) OpenFile(name string, flag filesystem.FileFlag, perm fs.FileMode) (filesystem.File, error) {
	return nil, fs.ErrInvalid
}

func (DefaultFileHandler) Stat(name string) (fs.FileInfo, error) {
	return nil, fs.ErrInvalid
}

func (DefaultFileHandler) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, fs.ErrInvalid
}

func (DefaultFileHandler) Mkdir(name string, perm fs.FileMode) (filesystem.DirFS, error) {
	return nil, fs.ErrInvalid
}

func (DefaultFileHandler) Readlink(name string) (string, error) {
	return "", fs.ErrInvalid
}

func (DefaultFileHandler) NewSocket(network socket.Network) (socket.Socket, error) {
	return nil, fs.ErrInvalid
}
