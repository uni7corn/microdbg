package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
)

type sysDirFile struct {
	File
	DirFS
}

type sysFileFS string
type sysDirFS string

func SysFileFS(name string) FS {
	return sysFileFS(name)
}

func SysDirFS(dir string) DirFS {
	return sysDirFS(dir)
}

func (f sysFileFS) Open(name string) (fs.File, error) {
	file, err := f.OpenFile(name, O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return file.(fs.File), nil
}

func (f sysFileFS) Stat(name string) (fs.FileInfo, error) {
	if name != "" {
		return nil, fs.ErrNotExist
	}
	return os.Stat(string(f))
}

func (f sysFileFS) OpenFile(name string, flag FileFlag, perm fs.FileMode) (File, error) {
	if name != "" {
		return nil, fs.ErrNotExist
	}
	return os.OpenFile(string(f), int(flag), perm)
}

func (d sysDirFS) Open(name string) (fs.File, error) {
	file, err := d.OpenFile(name, O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return file.(fs.File), nil
}

func (d sysDirFS) Sub(dir string) (fs.FS, error) {
	return d.sub(dir), nil
}

func (d sysDirFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(d.join(name))
}

func (d sysDirFS) OpenFile(name string, flag FileFlag, perm fs.FileMode) (File, error) {
	sysFile, err := os.OpenFile(d.join(name), int(flag), perm)
	if err != nil {
		return nil, err
	}
	info, err := sysFile.Stat()
	if err != nil || !info.IsDir() {
		return sysFile, nil
	}
	return &sysDirFile{sysFile, d.sub(name)}, nil
}

func (d sysDirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	_, file := filepath.Split(name)
	if file == "" {
		return os.ReadDir(string(d))
	}
	sub, err := d.Sub(name)
	if err != nil {
		return nil, err
	}
	return sub.(DirFS).ReadDir("")
}

func (d sysDirFS) Mkdir(name string, perm fs.FileMode) (DirFS, error) {
	pathname := d.join(name)
	err := os.MkdirAll(pathname, perm)
	if err != nil {
		return nil, err
	}
	return SysDirFS(pathname), nil
}

func (d sysDirFS) Readlink(name string) (string, error) {
	pathname := d.join(name)
	return os.Readlink(pathname)
}

func (d sysDirFS) join(name string) string {
	return filepath.Join(string(d), name)
}

func (d sysDirFS) sub(name string) DirFS {
	return SysDirFS(d.join(name))
}
