package socket

import (
	"io/fs"
	"time"
)

type info struct {
}

func (info) Name() string {
	return ""
}

func (info) Size() int64 {
	return 0
}

func (info) Mode() fs.FileMode {
	return fs.ModeSocket
}

func (info) ModTime() time.Time {
	return time.Now()
}

func (info) IsDir() bool {
	return false
}

func (info) Sys() any {
	return nil
}
