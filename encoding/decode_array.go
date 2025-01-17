package encoding

import (
	"unsafe"

	"github.com/modern-go/reflect2"
)

func decodeArray(typ reflect2.Type, bs int) (handler, structSize) {
	t := typ.Type1()
	count := t.Len()
	elemType := t.Elem()
	unmarshal, elemSize := decode(reflect2.Type2(elemType), bs)
	var size structSize
	for i := 0; i < count; i++ {
		size = size.Add(elemSize)
	}
	elemTotalSize := elemSize.Size()
	return func(stream Stream, ptr unsafe.Pointer) error {
		for i := 0; i < count; i++ {
			err := unmarshal(stream, ptr)
			if err != nil {
				return err
			}
			ptr = unsafe.Add(ptr, elemTotalSize)
		}
		return nil
	}, size
}

func decodeSlice(typ reflect2.Type, bs int) (handler, structSize) {
	unmarshal, elemSize := decode(reflect2.Type2(typ.Type1().Elem()), bs)
	elemTotalSize := elemSize.Size()
	return func(stream Stream, ptr unsafe.Pointer) error {
		subStream, err := stream.ReadStream()
		if err != nil {
			return err
		} else if subStream.Offset() == 0 {
			return nil
		}
		slice := (*sliceData)(ptr)
		ptr = slice.Data
		for i := 0; i < slice.Len; i++ {
			err = unmarshal(subStream, ptr)
			if err != nil {
				return err
			}
			ptr = unsafe.Add(ptr, elemTotalSize)
		}
		return nil
	}, structSize{bs}
}

func decodeString(bs int) (handler, structSize) {
	return func(stream Stream, ptr unsafe.Pointer) error {
		subStream, err := stream.ReadStream()
		if err != nil {
			return err
		} else if subStream.Offset() == 0 {
			return nil
		}
		str, err := subStream.ReadString()
		if err != nil {
			return err
		}
		slice := (*sliceData)(ptr)
		new := (*sliceData)(unsafe.Pointer(&str))
		*slice, *new = *new, *slice
		return nil
	}, structSize{bs, bs}
}
