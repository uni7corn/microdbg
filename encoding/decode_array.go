package encoding

import (
	"reflect"
	"unsafe"
)

func decodeArray(typ reflect.Type, bs int) (handler, structSize) {
	count := typ.Len()
	elemType := typ.Elem()
	if !checkCustom(elemType, bs) {
		size := make(structSize, count)
		elemSize := int(elemType.Size())
		for i := range size {
			size[i] = elemSize
		}
		totalSize := size.Size()
		return func(stream Stream, ptr unsafe.Pointer) error {
			_, err := stream.Read(unsafe.Slice((*byte)(ptr), totalSize))
			return err
		}, size
	}
	unmarshal, elemSize := decode(elemType, bs)
	size := make(structSize, 0, count*len(elemSize))
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

func decodeSlice(typ reflect.Type, bs int) (handler, structSize) {
	elemType := typ.Elem()
	if !checkCustom(elemType, bs) {
		elemSize := int(elemType.Size())
		return func(stream Stream, ptr unsafe.Pointer) error {
			subStream, err := stream.ReadStream()
			if err != nil {
				return err
			} else if subStream.Offset() == 0 {
				return nil
			}
			slice := (*sliceData)(ptr)
			_, err = stream.Read(unsafe.Slice((*byte)(slice.Data), elemSize*slice.Len))
			return err
		}, structSize{bs}
	}
	unmarshal, elemSize := decode(elemType, bs)
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
