package encoding

import (
	"reflect"
	"unsafe"
)

func decodeStruct(typ reflect.Type, bs int) (handler, structSize) {
	count := typ.NumField()
	size := make(structSize, 0, count)
	var offset uintptr
	var needCustom bool
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
			_, err := stream.Read(unsafe.Slice((*byte)(ptr), totalSize))
			return err
		}, size
	}
	size = size[:0]
	fields := make([]*structData, 0, count)
	for field := range rangeField(typ) {
		if field.Tag.Get("encoding") == "ignore" {
			continue
		}
		marshal, fieldSize := decodeFieldAlign(field.Type, bs, size.Size())
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

func decodeFieldAlign(typ reflect.Type, bs, offset int) (handler, structSize) {
	unmarshal, size := decode(typ, bs)
	addr := align(offset, size[0])
	if addr == offset {
		return unmarshal, size
	}
	pad := int(addr - offset)
	return func(stream Stream, ptr unsafe.Pointer) error {
		err := stream.Skip(pad)
		if err != nil {
			return err
		}
		return unmarshal(stream, ptr)
	}, append(structSize{pad}, size...)
}
