package encoding

import (
	"unsafe"

	"github.com/modern-go/reflect2"
)

type sliceData struct {
	Data unsafe.Pointer
	Len  int
}

func encodeArray(typ reflect2.Type, bs int) (handler, structSize) {
	t := typ.Type1()
	count := t.Len()
	marshal, elemSize := encode(reflect2.Type2(t.Elem()), bs)
	var size structSize
	for i := 0; i < count; i++ {
		size = size.Add(elemSize)
	}
	elemTotalSize := elemSize.Size()
	return func(stream Stream, ptr unsafe.Pointer) error {
		for i := 0; i < count; i++ {
			err := marshal(stream, ptr)
			if err != nil {
				return err
			}
			ptr = unsafe.Add(ptr, elemTotalSize)
		}
		return nil
	}, size
}

func encodeSlice(typ reflect2.Type, bs int) (handler, structSize) {
	marshal, elemSize := encode(reflect2.Type2(typ.Type1().Elem()), bs)
	elemTotalSize := elemSize.Size()
	return func(stream Stream, ptr unsafe.Pointer) error {
		slice := (*sliceData)(ptr)
		subStream, err := stream.WriteStream(elemTotalSize * slice.Len)
		if err != nil {
			return err
		}
		ptr = slice.Data
		for i := 0; i < slice.Len; i++ {
			err = marshal(subStream, ptr)
			if err != nil {
				return err
			}
			ptr = unsafe.Add(ptr, elemTotalSize)
		}
		return nil
	}, structSize{bs}
}

func encodeString(bs int) (handler, structSize) {
	return func(stream Stream, ptr unsafe.Pointer) error {
		slice := (*sliceData)(ptr)
		subStream, err := stream.WriteStream(slice.Len + 1)
		if err != nil {
			return err
		}
		return subStream.WriteString(unsafe.String((*byte)(slice.Data), slice.Len))
	}, structSize{bs, bs}
}
