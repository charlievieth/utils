package unsafe_slice

import (
	"reflect"
	"unsafe"
)

func Bytes(i interface{}) []byte {
	const (
		lenOff uintptr = 8
		capOff uintptr = 16
	)
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Slice {
		return nil
	}
	n := v.Len() * int(v.Type().Elem().Size())
	var b []byte
	*(*uintptr)(unsafe.Pointer((uintptr(unsafe.Pointer(&b))))) = v.Pointer()
	*(*int)(unsafe.Pointer((uintptr(unsafe.Pointer(&b)) + lenOff))) = n
	*(*int)(unsafe.Pointer((uintptr(unsafe.Pointer(&b)) + capOff))) = n
	p := make([]byte, len(b))
	copy(p, b)
	return p
}
