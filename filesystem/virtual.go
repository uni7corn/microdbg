package filesystem

import (
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type VirtualFS interface {
	ReadlinkFS
	Link(name string, handle FS) error
}

type fileFS struct {
	name    string
	mode    fs.FileMode
	modTime time.Time
	data    []byte
}

type dirFS struct {
	name    string
	mode    fs.FileMode
	modTime time.Time
	subs    sync.Map
}

type linkFS struct {
	name string
	fs   FS
}

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

type file struct {
	fs   *fileFS
	flag FileFlag
	off  int
}

type dir struct {
	fs   *dirFS
	read map[FS]struct{}
}

func NewVirtualFS() VirtualFS {
	return &dirFS{mode: fs.ModeDir}
}

func SoftLink(name string, fs FS) FS {
	return &linkFS{name: name, fs: fs}
}

func (f *fileFS) Open(name string) (fs.File, error) {
	return Open(f, name)
}

func (f *fileFS) Stat(name string) (fs.FileInfo, error) {
	if name != "" {
		return nil, fs.ErrInvalid
	}
	return &fileInfo{name: f.name, size: int64(len(f.data)), mode: f.mode, modTime: f.modTime}, nil
}

func (f *fileFS) OpenFile(name string, flag FileFlag, perm fs.FileMode) (File, error) {
	if name != "" {
		return nil, fs.ErrInvalid
	}
	if flag&O_EXCL != 0 {
		return nil, fs.ErrExist
	}
	var off int
	switch flag & (O_APPEND | O_TRUNC) {
	case O_APPEND:
		off = len(f.data)
	case O_TRUNC:
		f.data = nil
	}
	return &file{fs: f, flag: flag, off: off}, nil
}

func (d *dirFS) Open(name string) (fs.File, error) {
	return Open(d, name)
}

func (d *dirFS) Sub(dir string) (fs.FS, error) {
	first, other := split(dir)
	if first == "" {
		return nil, fs.ErrInvalid
	}
	value, ok := d.subs.Load(first)
	if !ok {
		return nil, fs.ErrNotExist
	} else if other == "" {
		return value.(fs.FS), nil
	}
	sub, ok := value.(fs.SubFS)
	if !ok {
		return nil, fs.ErrInvalid
	}
	return sub.Sub(other)
}

func (d *dirFS) Stat(name string) (fs.FileInfo, error) {
	dn, fn := filepath.Split(name)
	if fn == "" {
		return &fileInfo{name: d.name, mode: d.mode, modTime: d.modTime}, nil
	} else if dn != "" {
		sub, err := d.Sub(dn)
		if err != nil {
			return nil, err
		}
		return fs.Stat(sub, fn)
	}
	value, ok := d.subs.Load(fn)
	if !ok {
		return nil, fs.ErrNotExist
	}
	return fs.Stat(value.(fs.FS), "")
}

func (d *dirFS) OpenFile(name string, flag FileFlag, perm fs.FileMode) (File, error) {
	dn, fn := filepath.Split(name)
	if fn == "" {
		if flag != O_RDONLY {
			return nil, fs.ErrInvalid
		}
		return &dir{fs: d, read: make(map[FS]struct{})}, nil
	}
	var sub FS
	if dn == "" {
		value, ok := d.subs.Load(fn)
		if !ok {
			if flag&O_CREATE == 0 {
				return nil, fs.ErrNotExist
			}
			sub = &fileFS{name: fn, mode: perm, modTime: time.Now()}
		} else {
			sub = value.(FS)
		}
		fn = ""
	} else {
		value, err := d.Sub(dn)
		if err != nil {
			return nil, err
		}
		sub = value.(FS)
	}
	return sub.OpenFile(fn, flag, perm)
}

func (d *dirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "" {
		var arr []fs.DirEntry
		for _, value := range d.subs.Range {
			sub := value.(fs.FS)
			info, err := fs.Stat(sub, "")
			if err != nil {
				return nil, err
			}
			arr = append(arr, fs.FileInfoToDirEntry(info))
		}
		return arr, nil
	}
	sub, err := d.Sub(name)
	if err != nil {
		return nil, err
	}
	return sub.(DirFS).ReadDir("")
}

func (d *dirFS) Mkdir(name string, perm fs.FileMode) (DirFS, error) {
	first, other := split(name)
	if first == "" {
		return nil, fs.ErrInvalid
	}
	value, ok := d.subs.Load(first)
	var subDir DirFS
	if ok {
		subDir, ok = value.(DirFS)
		if !ok {
			return nil, fs.ErrInvalid
		}
	} else {
		subDir = &dirFS{name: first, mode: perm | fs.ModeDir, modTime: time.Now()}
		d.subs.Store(first, subDir)
	}
	if other == "" {
		return subDir, nil
	}
	return subDir.Mkdir(other, perm)
}

func (d *dirFS) Readlink(name string) (string, error) {
	dn, fn := filepath.Split(name)
	if fn == "" {
		return "", fs.ErrNotExist
	}
	if dn == "" {
		value, ok := d.subs.Load(fn)
		if !ok {
			return "", fs.ErrNotExist
		} else if link, ok := value.(*linkFS); ok {
			return link.name, nil
		}
	} else {
		value, err := d.Sub(dn)
		if err != nil {
			return "", err
		} else if vfs, ok := value.(VirtualFS); ok {
			return vfs.Readlink(fn)
		}
	}
	return "", fs.ErrInvalid
}

func (d *dirFS) Link(name string, handle FS) error {
	dir, file := filepath.Split(name)
	if dir != "" {
		sub, err := d.Mkdir(dir, fs.ModePerm)
		if err != nil {
			return err
		}
		link, ok := sub.(VirtualFS)
		if !ok {
			return fs.ErrInvalid
		}
		return link.Link(file, handle)
	}
	if _, ok := d.subs.LoadOrStore(file, handle); ok {
		return fs.ErrExist
	}
	return nil
}

func (l *linkFS) Open(name string) (fs.File, error) {
	if l.fs == nil {
		return nil, fs.ErrNotExist
	}
	return l.fs.Open(name)
}

func (l *linkFS) Sub(dir string) (fs.FS, error) {
	if sub, ok := l.fs.(fs.SubFS); ok {
		return sub.Sub(dir)
	}
	return nil, fs.ErrInvalid
}

func (l *linkFS) Stat(name string) (fs.FileInfo, error) {
	if stat, ok := l.fs.(fs.StatFS); ok {
		return stat.Stat(name)
	}
	return nil, fs.ErrInvalid
}

func (l *linkFS) OpenFile(name string, flag FileFlag, perm fs.FileMode) (File, error) {
	if l.fs == nil {
		return nil, fs.ErrNotExist
	}
	return l.fs.OpenFile(name, flag, perm)
}

func (l *linkFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if dir, ok := l.fs.(DirFS); ok {
		return dir.ReadDir(name)
	}
	return nil, fs.ErrInvalid
}

func (l *linkFS) Mkdir(name string, perm fs.FileMode) (DirFS, error) {
	if dir, ok := l.fs.(DirFS); ok {
		return dir.Mkdir(name, perm)
	}
	return nil, fs.ErrInvalid
}

func (l *linkFS) Readlink(name string) (string, error) {
	if vfs, ok := l.fs.(VirtualFS); ok {
		return vfs.Readlink(name)
	}
	return "", fs.ErrInvalid
}

func (l *linkFS) Link(name string, handle FS) error {
	if vfs, ok := l.fs.(VirtualFS); ok {
		return vfs.Link(name, handle)
	}
	return fs.ErrInvalid
}

func (fi fileInfo) Name() string {
	return fi.name
}

func (fi fileInfo) Size() int64 {
	return fi.size
}

func (fi fileInfo) Mode() fs.FileMode {
	return fi.mode
}

func (fi fileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi fileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

func (fi fileInfo) Sys() any {
	return nil
}

func (f *file) Close() error {
	return nil
}

func (f *file) Stat() (fs.FileInfo, error) {
	return &fileInfo{name: f.fs.name, size: int64(len(f.fs.data)), mode: f.fs.mode, modTime: f.fs.modTime}, nil
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	var off int
	switch whence {
	case io.SeekStart:
		off = int(offset)
	case io.SeekCurrent:
		off = f.off + int(offset)
	case io.SeekEnd:
		off = len(f.fs.data) + int(offset)
	default:
		return 0, fs.ErrInvalid
	}
	if off < 0 || off >= len(f.fs.data) {
		return 0, fs.ErrInvalid
	}
	f.off = off
	return int64(off), nil
}

func (f *file) Read(b []byte) (int, error) {
	if f.flag&O_WRONLY != 0 {
		return 0, fs.ErrInvalid
	}
	data := f.fs.data[f.off:]
	if len(data) == 0 {
		return 0, io.EOF
	}
	n := copy(b, data)
	f.off += n
	return n, nil
}

func (f *file) Write(b []byte) (int, error) {
	if f.flag&(O_WRONLY|O_RDWR) == 0 {
		return 0, fs.ErrInvalid
	}
	data := f.fs.data[f.off:]
	n := len(b)
	if len(data) >= n {
		n = copy(data, b)
	} else {
		f.fs.data = append(f.fs.data[:f.off], b...)
	}
	f.off += n
	return n, nil
}

func (d *dir) Close() error {
	return nil
}

func (d *dir) Stat() (fs.FileInfo, error) {
	return &fileInfo{name: d.fs.name, mode: d.fs.mode, modTime: d.fs.modTime}, nil
}

func (d *dir) OpenFile(name string, flag FileFlag, perm fs.FileMode) (File, error) {
	return d.fs.OpenFile(name, flag, perm)
}

func (d *dir) ReadDir(n int) ([]fs.DirEntry, error) {
	if n == 0 {
		return nil, nil
	}
	var arr []fs.DirEntry
	for _, value := range d.fs.subs.Range {
		sub := value.(FS)
		if _, ok := d.read[sub]; ok {
			continue
		}
		info, err := fs.Stat(sub, "")
		if err != nil {
			return nil, err
		}
		arr = append(arr, fs.FileInfoToDirEntry(info))
		d.read[sub] = struct{}{}
		n--
	}
	return arr, nil
}

func (d *dir) Mkdir(name string, perm fs.FileMode) error {
	dn, fn := filepath.Split(name)
	if dn != "" {
		return fs.ErrInvalid
	}
	sub := &dirFS{name: fn, mode: perm | fs.ModeDir, modTime: time.Now()}
	d.fs.subs.Store(fn, sub)
	return nil
}

func (d *dir) Readlink(name string) (string, error) {
	return d.fs.Readlink(name)
}

func split(pathname string) (string, string) {
	pathname = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(pathname)), "/")
	i := strings.Index(pathname, "/")
	if i == -1 {
		return pathname, ""
	}
	return pathname[:i], pathname[i+1:]
}
