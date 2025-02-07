package debugger

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/wnxd/microdbg/debugger"
	"github.com/wnxd/microdbg/filesystem"
	"github.com/wnxd/microdbg/socket"
)

type fileRef struct {
	file  filesystem.File
	count int64
}

type fileManager struct {
	handlers []debugger.FileHandler
	fd       int64
	fileRW   sync.RWMutex
	fileMap  map[int]filesystem.File
}

func (fm *fileManager) ctor() {
	fm.fd = 3
	fm.fileMap = map[int]filesystem.File{
		0: &fileRef{file: os.Stdin},
		1: &fileRef{file: os.Stdout},
		2: &fileRef{file: os.Stderr},
	}
}

func (fm *fileManager) dtor() {
	fm.handlers = nil
	fm.fileMap = nil
}

func (fm *fileManager) AddFileHandler(handler debugger.FileHandler) {
	fm.handlers = append(fm.handlers, handler)
}

func (fm *fileManager) RemoveFileHandler(handler debugger.FileHandler) {
	fm.handlers = slices.DeleteFunc(fm.handlers, func(h debugger.FileHandler) bool { return h == handler })
}

func (fm *fileManager) CreateFileDescriptor(file filesystem.File) int {
	fd := int(atomic.AddInt64(&fm.fd, 1))
	fm.fileRW.Lock()
	fm.fileMap[fd] = file
	fm.fileRW.Unlock()
	return int(fd)
}

func (fm *fileManager) CloseFileDescriptor(fd int) (filesystem.File, error) {
	fm.fileRW.Lock()
	defer fm.fileRW.Unlock()
	if file, ok := fm.fileMap[fd]; ok {
		delete(fm.fileMap, fd)
		return file, nil
	}
	return nil, fs.ErrNotExist
}

func (fm *fileManager) GetFile(fd int) (filesystem.File, error) {
	fm.fileRW.RLock()
	defer fm.fileRW.RUnlock()
	if file, ok := fm.fileMap[fd]; ok {
		return file, nil
	}
	return nil, fs.ErrNotExist
}

func (fm *fileManager) DupFile(fd int) (int, error) {
	fm.fileRW.Lock()
	defer fm.fileRW.Unlock()
	file, ok := fm.fileMap[fd]
	if !ok {
		return -1, fs.ErrNotExist
	}
	ref, ok := file.(*fileRef)
	if ok {
		atomic.AddInt64(&ref.count, 1)
	} else {
		ref = &fileRef{file: file, count: 2}
		fm.fileMap[fd] = ref
	}
	newfd := int(atomic.AddInt64(&fm.fd, 1))
	fm.fileMap[newfd] = ref
	return newfd, nil
}

func (fm *fileManager) Dup2File(oldfd, newfd int) error {
	if oldfd == newfd {
		return nil
	} else if newfd < 3 {
		return nil
	}
	fm.fileRW.Lock()
	defer fm.fileRW.Unlock()
	file, ok := fm.fileMap[oldfd]
	if !ok {
		return fs.ErrNotExist
	}
	if old, ok := fm.fileMap[newfd]; ok {
		old.Close()
	}
	ref, ok := file.(*fileRef)
	if ok {
		atomic.AddInt64(&ref.count, 1)
	} else {
		ref = &fileRef{file: file, count: 2}
		fm.fileMap[oldfd] = ref
	}
	fm.fileMap[newfd] = ref
	return nil
}

func (fm *fileManager) GetFS() filesystem.FS {
	return fm
}

func (fm *fileManager) Open(name string) (fs.File, error) {
	return filesystem.Open(fm, name)
}

func (fm *fileManager) OpenFile(name string, flag filesystem.FileFlag, perm fs.FileMode) (filesystem.File, error) {
	name = filepath.ToSlash(filepath.Join("/", name))
	for _, handler := range fm.handlers {
		f, err := handler.OpenFile(name, flag, perm)
		if err == nil {
			return f, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (fm *fileManager) Stat(name string) (fs.FileInfo, error) {
	name = filepath.ToSlash(filepath.Join("/", name))
	for _, handler := range fm.handlers {
		fi, err := handler.Stat(name)
		if err == nil {
			return fi, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (fm *fileManager) ReadDir(name string) ([]fs.DirEntry, error) {
	name = filepath.ToSlash(filepath.Join("/", name))
	for _, handler := range fm.handlers {
		list, err := handler.ReadDir(name)
		if err == nil {
			return list, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (fm *fileManager) Mkdir(name string, perm fs.FileMode) (filesystem.DirFS, error) {
	name = filepath.ToSlash(filepath.Join("/", name))
	for _, handler := range fm.handlers {
		dir, err := handler.Mkdir(name, perm)
		if err == nil {
			return dir, nil
		}
	}
	return nil, fs.ErrInvalid
}

func (fm *fileManager) Readlink(name string) (string, error) {
	name = filepath.ToSlash(filepath.Join("/", name))
	for _, handler := range fm.handlers {
		path, err := handler.Readlink(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fs.ErrInvalid
}

func (fm *fileManager) NewSocket(network socket.Network) (socket.Socket, error) {
	for _, handler := range fm.handlers {
		s, err := handler.NewSocket(network)
		if err == nil {
			return s, nil
		}
	}
	return socket.New(network), nil
}

func (f *fileRef) Close() error {
	i := atomic.AddInt64(&f.count, -1)
	if i > 0 {
		return nil
	} else if i < 0 {
		return fs.ErrClosed
	}
	return f.file.Close()
}

func (f *fileRef) Stat() (fs.FileInfo, error) {
	return f.file.Stat()
}

func (f *fileRef) Read(b []byte) (int, error) {
	if r, ok := f.file.(filesystem.ReadFile); ok {
		return r.Read(b)
	}
	return 0, errors.ErrUnsupported
}

func (f *fileRef) Write(b []byte) (int, error) {
	if w, ok := f.file.(filesystem.WriteFile); ok {
		return w.Write(b)
	}
	return 0, errors.ErrUnsupported
}

func (f *fileRef) ReadDir(n int) ([]fs.DirEntry, error) {
	if dir, ok := f.file.(filesystem.DirFile); ok {
		return dir.ReadDir(n)
	}
	return nil, errors.ErrUnsupported
}

func (f *fileRef) OpenFile(name string, flag filesystem.FileFlag, perm fs.FileMode) (filesystem.File, error) {
	if dir, ok := f.file.(filesystem.Dir); ok {
		return dir.OpenFile(name, flag, perm)
	}
	return nil, errors.ErrUnsupported
}

func (f *fileRef) Mkdir(name string, perm fs.FileMode) error {
	if dir, ok := f.file.(filesystem.Dir); ok {
		return dir.Mkdir(name, perm)
	}
	return errors.ErrUnsupported
}

func (f *fileRef) Control(op int, arg any) error {
	if ctl, ok := f.file.(filesystem.ControlFile); ok {
		return ctl.Control(op, arg)
	}
	return errors.ErrUnsupported
}
