package encoding

import (
	"reflect"
	"sync"
	"unsafe"
)

var decodeProcess sync.Map

func DecodeSize(blockSize int, val any) int {
	if getPtr(val) == nil {
		return blockSize
	}
	typ := reflect.TypeOf(val)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return getUnmarshalData(typ, blockSize).size
}

func Decode(stream Stream, val any) error {
	bs := stream.BlockSize()
	ptr := getPtr(val)
	if ptr == nil {
		return stream.Skip(bs)
	}
	typ := reflect.TypeOf(val)
	switch typ.Kind() {
	case reflect.Pointer:
		typ = typ.Elem()
	case reflect.Slice, reflect.String:
		stream = fakeStream{stream}
	case reflect.Struct:
		if typ.NumField() == 1 && typ.Field(0).Type.Kind() == reflect.Pointer {
			return getUnmarshalData(typ, bs).handler(stream, unsafe.Pointer(&ptr))
		}
	}
	return getUnmarshalData(typ, bs).handler(stream, ptr)
}

func getUnmarshalData(typ reflect.Type, bs int) *handlerData {
	key := [2]uintptr{uintptr(bs), getRType(typ)}
	var data *handlerData
	if v, ok := decodeProcess.Load(key); ok {
		data = v.(*handlerData)
	} else {
		unmarshal, size := decode(typ, bs)
		data = &handlerData{unmarshal, size.Size()}
		decodeProcess.Store(key, data)
	}
	return data
}

func decode(typ reflect.Type, bs int) (handler, structSize) {
	switch typ.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Complex64, reflect.Complex128:
		size := int(typ.Size())
		return func(stream Stream, ptr unsafe.Pointer) error {
			_, err := stream.Read(unsafe.Slice((*byte)(ptr), size))
			return err
		}, structSize{size}
	case reflect.Float32:
		return func(stream Stream, ptr unsafe.Pointer) error {
			f, err := stream.ReadFloat()
			if err == nil {
				*(*float32)(ptr) = f
			}
			return err
		}, structSize{4}
	case reflect.Float64:
		return func(stream Stream, ptr unsafe.Pointer) error {
			d, err := stream.ReadDouble()
			if err == nil {
				*(*float64)(ptr) = d
			}
			return err
		}, structSize{8}
	case reflect.Int, reflect.Uint, reflect.Uintptr, reflect.UnsafePointer:
		size := int(typ.Size())
		var pad int
		if size > bs {
			size = bs
		} else if size < bs {
			pad = int(bs - size)
		}
		return func(stream Stream, ptr unsafe.Pointer) error {
			_, err := stream.Read(unsafe.Slice((*byte)(ptr), size))
			if err != nil {
				return err
			} else if pad > 0 {
				return stream.Skip(pad)
			}
			return nil
		}, structSize{bs}
	case reflect.Array:
		return decodeArray(typ, bs)
	case reflect.Pointer:
		return decodePointer(typ.Elem(), bs)
	case reflect.Slice:
		return decodeSlice(typ, bs)
	case reflect.String:
		return decodeString(bs)
	case reflect.Struct:
		return decodeStruct(typ, bs)
	}
	panic("Unsupported Type")
}

func decodePointer(typ reflect.Type, bs int) (handler, structSize) {
	unmarshal, _ := decode(typ, bs)
	return func(stream Stream, ptr unsafe.Pointer) error {
		subStream, err := stream.ReadStream()
		if err != nil {
			return err
		} else if subStream.Offset() == 0 {
			return nil
		}
		elemPtr := *(*unsafe.Pointer)(ptr)
		if elemPtr == nil {
			elemPtr = unsafe_New(getRType(typ))
			*(*unsafe.Pointer)(ptr) = elemPtr
		}
		return unmarshal(subStream, elemPtr)
	}, structSize{bs}
}

//go:linkname unsafe_New reflect.unsafe_New
func unsafe_New(rtype uintptr) unsafe.Pointer
