package debugger

import (
	"io"
)

type Buffer []byte

func (buf *Buffer) ReadAt(b []byte, off int64) (n int, err error) {
	if int(off) >= len(*buf) {
		return 0, io.EOF
	}
	return copy(b, (*buf)[off:]), nil
}

func (buf *Buffer) WriteAt(b []byte, off int64) (n int, err error) {
	if end := len(b) + int(off); end > len(*buf) {
		*buf = append(*buf, make([]byte, end-len(*buf))...)
	}
	return copy((*buf)[off:], b), nil
}
