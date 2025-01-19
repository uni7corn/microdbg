package debugger

import (
	"unsafe"
)

func ReadPtrRaw[V any](raw []byte) V {
	return *(*V)(unsafe.Pointer(unsafe.SliceData(raw)))
}

func ToPtrRaw[S any](ptr *S) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), unsafe.Sizeof(*ptr))
}

func ConvertRaw[S any](src []S) []byte {
	size := uintptr(len(src)) * Sizeof[S]()
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(src))), size)
}

func Sizeof[V any]() uintptr {
	var v V
	return unsafe.Sizeof(v)
}

func GetPtr(v any) unsafe.Pointer {
	return (*struct{ _, data unsafe.Pointer })(unsafe.Pointer(&v)).data
}
