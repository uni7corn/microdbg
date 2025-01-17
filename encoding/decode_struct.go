package encoding

import (
	"reflect"
	"unsafe"

	"github.com/modern-go/reflect2"
)

func decodeStruct(typ reflect2.Type, bs int) (handler, structSize) {
	t := typ.Type1()
	count := t.NumField()
	size := make(structSize, 0, count)
	var offset uintptr
	var needMarshal bool
	for field := range rangeField(t) {
		if field.Tag.Get("encoding") == "ignore" {
			needMarshal = true
			break
		}
		switch field.Type.Kind() {
		case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		case reflect.Uintptr, reflect.UnsafePointer:
			needMarshal = int(field.Type.Size()) != bs
		default:
			needMarshal = true
		}
		if needMarshal {
			break
		} else if s := field.Offset - offset; s != 0 {
			size = append(size, int(s))
		}
		offset = field.Offset
	}
	if !needMarshal {
		size = append(size, int(t.Size()-offset))
		totalSize := size.Size()
		return func(stream Stream, ptr unsafe.Pointer) error {
			_, err := stream.Read(unsafe.Slice((*byte)(ptr), totalSize))
			return err
		}, size
	}
	size = size[:0]
	fields := make([]*structData, 0, count)
	for field := range rangeField(t) {
		if field.Tag.Get("encoding") == "ignore" {
			continue
		}
		marshal, fieldSize := decodeFieldAlign(reflect2.Type2(field.Type), bs, size.Size())
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

func decodeFieldAlign(typ reflect2.Type, bs, offset int) (handler, structSize) {
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
