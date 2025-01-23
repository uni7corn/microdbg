package encoding

import (
	"iter"
	"reflect"
	"unsafe"
)

type structData struct {
	handler handler
	offset  int
}

func encodeStruct(typ reflect.Type, bs int) (handler, structSize) {
	count := typ.NumField()
	size := make(structSize, 0, count)
	var (
		offset     uintptr
		needCustom bool
	)
	for field := range rangeField(typ) {
		if field.Tag.Get("encoding") == "ignore" {
			needCustom = true
			break
		}
		if needCustom = checkCustom(field.Type, bs); needCustom {
			break
		} else if s := field.Offset - offset; s != 0 {
			size = append(size, int(s))
		}
		offset = field.Offset
	}
	if !needCustom {
		size = append(size, int(typ.Size()-offset))
		totalSize := size.Size()
		return func(stream Stream, ptr unsafe.Pointer) error {
			_, err := stream.Write(unsafe.Slice((*byte)(ptr), totalSize))
			return err
		}, size
	}
	size = size[:0]
	fields := make([]*structData, 0, count)
	for field := range rangeField(typ) {
		if field.Tag.Get("encoding") == "ignore" {
			continue
		}
		marshal, fieldSize := encodeFieldAlign(field.Type, bs, size.Size())
		size = size.Add(fieldSize)
		fields = append(fields, &structData{marshal, int(field.Offset)})
	}
	var maxSize int
	for _, s := range size {
		maxSize = max(maxSize, s)
	}
	totalSize := size.Size()
	pad := align(totalSize, maxSize) - totalSize
	if pad > 0 {
		size = append(size, pad)
	}
	return func(stream Stream, ptr unsafe.Pointer) error {
		for _, data := range fields {
			err := data.handler(stream, unsafe.Add(ptr, data.offset))
			if err != nil {
				return err
			}
		}
		if pad > 0 {
			return stream.Skip(pad)
		}
		return nil
	}, size
}

func encodeFieldAlign(typ reflect.Type, bs, offset int) (handler, structSize) {
	marshal, size := encode(typ, bs)
	addr := align(offset, size[0])
	if addr == offset {
		return marshal, size
	}
	pad := int(addr - offset)
	return func(stream Stream, ptr unsafe.Pointer) error {
		err := stream.Skip(pad)
		if err != nil {
			return err
		}
		return marshal(stream, ptr)
	}, append(structSize{pad}, size...)
}

func rangeField(typ reflect.Type) iter.Seq[reflect.StructField] {
	return func(yield func(reflect.StructField) bool) {
		count := typ.NumField()
		for i := 0; i < count; i++ {
			if !yield(typ.Field(i)) {
				break
			}
		}
	}
}

func align(a, b int) int {
	return (a + b - 1) &^ (b - 1)
}
