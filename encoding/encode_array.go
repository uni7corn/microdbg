package encoding

import (
	"reflect"
	"unsafe"
)

type sliceData struct {
	Data unsafe.Pointer
	Len  int
}

func encodeArray(typ reflect.Type, bs int) (handler, structSize) {
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
			_, err := stream.Write(unsafe.Slice((*byte)(ptr), totalSize))
			return err
		}, size
	}
	marshal, elemSize := encode(elemType, bs)
	size := make(structSize, 0, count*len(elemSize))
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

func encodeSlice(typ reflect.Type, bs int) (handler, structSize) {
	elemType := typ.Elem()
	if !checkCustom(elemType, bs) {
		elemSize := int(elemType.Size())
		return func(stream Stream, ptr unsafe.Pointer) error {
			slice := (*sliceData)(ptr)
			totalSize := elemSize * slice.Len
			subStream, err := stream.WriteStream(totalSize)
			if err != nil {
				return err
			}
			_, err = subStream.Write(unsafe.Slice((*byte)(slice.Data), totalSize))
			return err
		}, structSize{bs}
	}
	marshal, elemSize := encode(elemType, bs)
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

func checkCustom(typ reflect.Type, bs int) bool {
	switch typ.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return false
	case reflect.Int, reflect.Uint, reflect.Uintptr, reflect.UnsafePointer:
		return int(typ.Size()) != bs
	default:
		return true
	}
}
