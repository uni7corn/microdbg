package encoding

import (
	"reflect"
	"sync"
	"unsafe"
)

type handler = func(Stream, unsafe.Pointer) error

type handlerData struct {
	handler handler
	size    int
}

var (
	encodeProcess sync.Map
	padNull       [8]byte
)

func EncodeSize(blockSize int, val any) int {
	if getPtr(val) == nil {
		return blockSize
	}
	return getMarshalData(reflect.TypeOf(val), blockSize).size
}

func Encode(stream Stream, val any) error {
	bs := stream.BlockSize()
	ptr := getPtr(val)
	if ptr == nil {
		_, err := stream.Write(padNull[:bs])
		return err
	}
	typ := reflect.TypeOf(val)
	handler := getMarshalData(typ, bs).handler
	switch typ.Kind() {
	case reflect.Pointer:
		return handler(stream, unsafe.Pointer(&ptr))
	case reflect.Struct:
		if typ.NumField() == 1 && typ.Field(0).Type.Kind() == reflect.Pointer {
			return handler(stream, unsafe.Pointer(&ptr))
		}
	}
	return handler(stream, ptr)
}

func getMarshalData(typ reflect.Type, bs int) *handlerData {
	key := [2]uintptr{uintptr(bs), getRType(typ)}
	var data *handlerData
	if v, ok := encodeProcess.Load(key); ok {
		data = v.(*handlerData)
	} else {
		marshal, size := encode(typ, bs)
		data = &handlerData{marshal, size.Size()}
		encodeProcess.Store(key, data)
	}
	return data
}

func encode(typ reflect.Type, bs int) (handler, structSize) {
	switch typ.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Complex64, reflect.Complex128:
		size := int(typ.Size())
		return func(stream Stream, ptr unsafe.Pointer) error {
			_, err := stream.Write(unsafe.Slice((*byte)(ptr), size))
			return err
		}, structSize{size}
	case reflect.Float32:
		return func(stream Stream, ptr unsafe.Pointer) error {
			return stream.WriteFloat(*(*float32)(ptr))
		}, structSize{4}
	case reflect.Float64:
		return func(stream Stream, ptr unsafe.Pointer) error {
			return stream.WriteDouble(*(*float64)(ptr))
		}, structSize{8}
	case reflect.Int, reflect.Uint, reflect.Uintptr, reflect.UnsafePointer:
		size := int(typ.Size())
		var pad int
		if size > bs {
			size = bs
		} else if size < bs {
			pad = bs - size
		}
		return func(stream Stream, ptr unsafe.Pointer) error {
			_, err := stream.Write(unsafe.Slice((*byte)(ptr), size))
			if err != nil {
				return err
			} else if pad > 0 {
				_, err = stream.Write(padNull[:pad])
				return err
			}
			return nil
		}, structSize{bs}
	case reflect.Array:
		return encodeArray(typ, bs)
	// case reflect.Interface:
	case reflect.Pointer:
		return encodePointer(typ.Elem(), bs)
	case reflect.Slice:
		return encodeSlice(typ, bs)
	case reflect.String:
		return encodeString(bs)
	case reflect.Struct:
		return encodeStruct(typ, bs)
	}
	panic("Unsupported Type")
}

func encodePointer(typ reflect.Type, bs int) (handler, structSize) {
	marshal, elemSize := encode(typ, bs)
	totalSize := elemSize.Size()
	return func(stream Stream, ptr unsafe.Pointer) error {
		if ptr == nil {
			return stream.Skip(bs)
		}
		subStream, err := stream.WriteStream(totalSize)
		if err != nil {
			return err
		}
		return marshal(subStream, *(*unsafe.Pointer)(ptr))
	}, structSize{bs}
}

func getRType(typ reflect.Type) uintptr {
	return (*struct{ _, data uintptr })(unsafe.Pointer(&typ)).data
}

func getPtr(v any) unsafe.Pointer {
	return (*struct{ _, data unsafe.Pointer })(unsafe.Pointer(&v)).data
}
