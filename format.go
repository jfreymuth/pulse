package pulse

import (
	"reflect"
	"unsafe"

	"github.com/jfreymuth/pulse/proto"
)

var formatI16, formatI32, formatF32 byte

func init() {
	i := uint16(1)
	littleEndian := *(*byte)(unsafe.Pointer(&i)) == 1
	if littleEndian {
		formatI16 = proto.FormatInt16LE
		formatI32 = proto.FormatInt32LE
		formatF32 = proto.FormatFloat32LE
	} else {
		formatI16 = proto.FormatInt16BE
		formatI32 = proto.FormatInt32BE
		formatF32 = proto.FormatFloat32BE
	}
}

func int16Slice(s []byte) []int16 {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&s))
	return *(*[]int16)(unsafe.Pointer(&reflect.SliceHeader{Data: h.Data, Len: h.Len / 2, Cap: h.Len / 2}))
}

func int32Slice(s []byte) []int32 {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&s))
	return *(*[]int32)(unsafe.Pointer(&reflect.SliceHeader{Data: h.Data, Len: h.Len / 4, Cap: h.Len / 4}))
}

func float32Slice(s []byte) []float32 {
	h := *(*reflect.SliceHeader)(unsafe.Pointer(&s))
	return *(*[]float32)(unsafe.Pointer(&reflect.SliceHeader{Data: h.Data, Len: h.Len / 4, Cap: h.Len / 4}))
}
