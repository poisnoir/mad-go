package mad

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sort"
	"unsafe"
)

type Mad[T any] struct {
	encoder         func(unsafe.Pointer, *[]byte)
	decoder         func(unsafe.Pointer, *[]byte) error
	sizefunc        func(pointer unsafe.Pointer) int
	fixedFieldsSize int
	buffer          []byte
	code            string
}

func (m *Mad[T]) Code() string {
	return m.code
}

func NewMad[T any]() (*Mad[T], error) {
	var zero T

	m := &Mad[T]{}

	encFn, decFn, sizefn, err, hash := generateFuncs(reflect.TypeOf(zero))
	if err != nil {
		return nil, err
	}

	m.encoder = encFn
	m.decoder = decFn
	m.sizefunc = sizefn
	m.code = hash
	return m, nil
}

func (m *Mad[T]) GetRequiredSize(input *T) int {
	return m.sizefunc(unsafe.Pointer(input))
}

func (m *Mad[T]) Encode(input *T, output []byte) (err error) {
	if len(output) < m.sizefunc(unsafe.Pointer(input)) {
		return fmt.Errorf("output buffer too small")
	}
	m.encoder(unsafe.Pointer(input), &output)
	return nil
}

func (m *Mad[T]) Decode(input []byte, output *T) (err error) {
	return m.decoder(unsafe.Pointer(output), &input)
}

func generateFuncs(typ reflect.Type) (func(unsafe.Pointer, *[]byte), func(unsafe.Pointer, *[]byte) error, func(unsafe.Pointer) int, error, string) {

	var enc func(unsafe.Pointer, *[]byte)
	var dec func(unsafe.Pointer, *[]byte) error
	var size func(pointer unsafe.Pointer) int
	var code string
	var err error

	// Check for nil type (happens with interface{})
	if typ == nil {
		return nil, nil, nil, fmt.Errorf("unsupported type: <nil>, please refer to documentation for supported types"), ""
	}

	switch typ.Kind() {
	case reflect.Int8, reflect.Bool, reflect.Uint8:
		enc, dec, size, code = byteStrat()
	case reflect.Int16, reflect.Uint16:
		enc, dec, size, code = twoByteStrat()
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		enc, dec, size, code = fourByteStrat()
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		enc, dec, size, code = eightByteStrat()
	case reflect.String:
		enc, dec, size, code = stringStrat()
	case reflect.Struct:
		enc, dec, size, err, code = structStrat(typ)
	case reflect.Array:
		enc, dec, size, err, code = arrStrat(typ)
	// Todo: Current implementation use slice header
	// I am not sure how safe it is since it is deprecated.
	// case reflect.Slice:
	// 	enc, dec, size, err = sliceStrat(typ)
	// I have feeling if I move to map I see the same issues so for now I pause that
	default:
		err = fmt.Errorf("unsupported type: %v, please refer to documentation for supported types", typ)
	}

	if err != nil {
		return nil, nil, nil, err, ""
	}

	return enc, dec, size, nil, code
}

func byteStrat() (func(unsafe.Pointer, *[]byte), func(unsafe.Pointer, *[]byte) error, func(unsafe.Pointer) int, string) {
	return func(input unsafe.Pointer, buffer *[]byte) {
			(*buffer)[0] = *(*byte)(input)
			*buffer = (*buffer)[1:]
		},
		func(output unsafe.Pointer, buffer *[]byte) error {
			if len(*buffer) < 1 {
				return fmt.Errorf("buffer too small")
			}
			*(*byte)(output) = (*buffer)[0]
			*buffer = (*buffer)[1:]
			return nil
		}, func(unsafe.Pointer) int {
			return 1
		}, "0"
}

func twoByteStrat() (func(unsafe.Pointer, *[]byte), func(unsafe.Pointer, *[]byte) error, func(unsafe.Pointer) int, string) {
	return func(pointer unsafe.Pointer, buffer *[]byte) {
			binary.BigEndian.PutUint16((*buffer)[0:2], *(*uint16)(pointer))
			*buffer = (*buffer)[2:]
		}, func(output unsafe.Pointer, buffer *[]byte) error {
			if len(*buffer) < 2 {
				return fmt.Errorf("buffer too small")
			}
			*(*uint16)(output) = binary.BigEndian.Uint16((*buffer)[0:2])
			*buffer = (*buffer)[2:]
			return nil
		}, func(unsafe.Pointer) int {
			return 2
		}, "1"
}

func fourByteStrat() (func(unsafe.Pointer, *[]byte), func(unsafe.Pointer, *[]byte) error, func(unsafe.Pointer) int, string) {
	return func(pointer unsafe.Pointer, buffer *[]byte) {
			binary.BigEndian.PutUint32((*buffer)[0:4], *(*uint32)(pointer))
			*buffer = (*buffer)[4:]
		}, func(output unsafe.Pointer, buffer *[]byte) error {
			if len(*buffer) < 4 {
				return fmt.Errorf("buffer too small")
			}
			*(*uint32)(output) = binary.BigEndian.Uint32((*buffer)[0:4])
			*buffer = (*buffer)[4:]
			return nil
		}, func(unsafe.Pointer) int {
			return 4
		}, "2"
}

func eightByteStrat() (func(unsafe.Pointer, *[]byte), func(unsafe.Pointer, *[]byte) error, func(unsafe.Pointer) int, string) {
	return func(pointer unsafe.Pointer, buffer *[]byte) {
			binary.BigEndian.PutUint64((*buffer)[0:8], *(*uint64)(pointer))
			*buffer = (*buffer)[8:]
		}, func(output unsafe.Pointer, buffer *[]byte) error {
			if len(*buffer) < 8 {
				return fmt.Errorf("buffer too small")
			}
			*(*uint64)(output) = binary.BigEndian.Uint64((*buffer)[0:8])
			*buffer = (*buffer)[8:]
			return nil
		}, func(unsafe.Pointer) int {
			return 8
		}, "3"
}

func stringStrat() (func(unsafe.Pointer, *[]byte), func(unsafe.Pointer, *[]byte) error, func(unsafe.Pointer) int, string) {
	return func(input unsafe.Pointer, buffer *[]byte) {
			str := *(*string)(input)
			n := len(str)
			binary.BigEndian.PutUint32((*buffer)[0:4], uint32(n))
			copy((*buffer)[4:], str)
			*buffer = (*buffer)[n+4:]
		}, func(output unsafe.Pointer, buffer *[]byte) error {
			if len(*buffer) < 4 {
				return fmt.Errorf("buffer too small")
			}
			n := binary.BigEndian.Uint32((*buffer)[0:4])
			*buffer = (*buffer)[4:]
			if len(*buffer) < int(n) {
				return fmt.Errorf("buffer too small")
			}
			*(*string)(output) = string((*buffer)[:n])
			*buffer = (*buffer)[n:]
			return nil
		}, func(input unsafe.Pointer) int {
			return len(*(*string)(input)) + 4
		}, "4"
}

func arrStrat(t reflect.Type) (func(unsafe.Pointer, *[]byte), func(unsafe.Pointer, *[]byte) error, func(unsafe.Pointer) int, error, string) {
	elementType := t.Elem()
	encElemFn, decElemFn, sizeElemFn, err, code := generateFuncs(elementType)
	if err != nil {
		return nil, nil, nil, err, ""
	}

	arrLen := t.Len()
	elementSize := elementType.Size()

	return func(pointer unsafe.Pointer, buffer *[]byte) {
			for i := 0; i < arrLen; i++ {
				itemPtr := unsafe.Add(pointer, uintptr(i)*elementSize)
				encElemFn(itemPtr, buffer)
			}
		}, func(pointer unsafe.Pointer, buffer *[]byte) error {

			for i := 0; i < arrLen; i++ {
				itemPtr := unsafe.Add(pointer, uintptr(i)*elementSize)
				err := decElemFn(itemPtr, buffer)
				if err != nil {
					return err
				}
			}
			return nil
		}, func(pointer unsafe.Pointer) int {
			total := 0
			for i := 0; i < arrLen; i++ {
				itemPtr := unsafe.Add(pointer, uintptr(i)*elementSize)
				total += sizeElemFn(itemPtr)
			}
			return total
		}, nil, "5" + code
}

func sliceStrat(t reflect.Type) (func(unsafe.Pointer, *[]byte), func(unsafe.Pointer, *[]byte) error, func(unsafe.Pointer) int, error, string) {

	elementType := t.Elem()
	encElemFn, decElemFn, sizeElemFn, err, code := generateFuncs(elementType)
	if err != nil {
		return nil, nil, nil, err, ""
	}

	elemSize := elementType.Size()

	return func(pointer unsafe.Pointer, buffer *[]byte) {
			base := unsafe.Pointer((*reflect.SliceHeader)(pointer).Data)
			sliceLen := (*reflect.SliceHeader)(pointer).Len
			binary.BigEndian.PutUint32((*buffer)[0:4], (uint32)(sliceLen))
			*buffer = (*buffer)[4:]
			itemPtr := base
			for i := 0; i < sliceLen; i++ {
				itemPtr = unsafe.Add(base, uintptr(i)*elemSize)
				encElemFn(itemPtr, buffer)
			}
		}, func(pointer unsafe.Pointer, buffer *[]byte) error {
			if len(*buffer) < 4 {
				return fmt.Errorf("buffer too small")
			}
			incomingDataLen := binary.BigEndian.Uint32((*buffer)[0:4])
			*buffer = (*buffer)[4:]
			newSlice := reflect.MakeSlice(t, int(incomingDataLen), int(incomingDataLen))
			*(*reflect.SliceHeader)(pointer) = *(*reflect.SliceHeader)(unsafe.Pointer(newSlice.UnsafeAddr()))
			base := unsafe.Pointer((*reflect.SliceHeader)(pointer).Data)
			itemPtr := base
			for i := uint32(0); i < incomingDataLen; i++ {
				itemPtr = unsafe.Add(base, uintptr(i)*elemSize)
				err := decElemFn(itemPtr, buffer)
				if err != nil {
					return err
				}
			}

			return nil
		}, func(pointer unsafe.Pointer) int {
			total := 4
			sliceLen := (*reflect.SliceHeader)(pointer).Len
			base := unsafe.Pointer((*reflect.SliceHeader)(pointer).Data)
			itemPtr := base
			for i := 0; i < sliceLen; i++ {
				itemPtr = unsafe.Add(base, uintptr(i)*elemSize)
				total += sizeElemFn(itemPtr)
			}
			return total
		}, nil, "6" + code
}

func structStrat(t reflect.Type) (func(unsafe.Pointer, *[]byte), func(unsafe.Pointer, *[]byte) error, func(unsafe.Pointer) int, error, string) {

	type fieldMeta struct {
		name   string
		offset uintptr
		typ    reflect.Type
		enc    func(unsafe.Pointer, *[]byte)
		dec    func(unsafe.Pointer, *[]byte) error
		size   func(unsafe.Pointer) int
		code   string
	}

	var fields []fieldMeta
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		encFn, decFn, sizeFn, err, code := generateFuncs(f.Type)
		if err != nil {
			return nil, nil, nil, err, ""
		}
		fields = append(fields, fieldMeta{
			offset: f.Offset, // This is the byte-distance from the struct start
			name:   f.Name,
			enc:    encFn,
			dec:    decFn,
			size:   sizeFn,
			code:   code,
		})
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].name < fields[j].name
	})

	// Generate the code for the struct
	code := "7"
	for i := 0; i < t.NumField(); i++ {
		code += fields[i].code
	}

	return func(pointer unsafe.Pointer, buffer *[]byte) {
			for _, field := range fields {
				fieldAddr := unsafe.Add(pointer, field.offset)
				field.enc(fieldAddr, buffer)
			}
		}, func(pointer unsafe.Pointer, buffer *[]byte) error {
			for _, field := range fields {
				fieldAddr := unsafe.Pointer(uintptr(pointer) + field.offset)
				if err := field.dec(fieldAddr, buffer); err != nil {
					return err
				}
			}
			return nil
		}, func(pointer unsafe.Pointer) int {
			total := 0
			for _, field := range fields {
				fieldAddr := unsafe.Pointer(uintptr(pointer) + field.offset)
				total += field.size(fieldAddr)
			}
			return total
		}, nil, code
}
