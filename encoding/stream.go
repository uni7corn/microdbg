package encoding

type Stream interface {
	BlockSize() int
	Offset() uint64
	Skip(int) error
	Read([]byte) (int, error)
	ReadFloat() (float32, error)
	ReadDouble() (float64, error)
	ReadString() (string, error)
	ReadStream() (Stream, error)
	Write([]byte) (int, error)
	WriteFloat(float32) error
	WriteDouble(float64) error
	WriteString(string) error
	WriteStream(int) (Stream, error)
}

type fakeStream struct {
	Stream
}

func (s fakeStream) ReadStream() (Stream, error) {
	return s.Stream, nil
}
