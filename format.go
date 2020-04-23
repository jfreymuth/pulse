package pulse

import (
	"io"
	"reflect"
	"unsafe"

	"github.com/jfreymuth/pulse/proto"
)

// A Reader provides audio data in a specific format.
type Reader interface {
	io.Reader
	Format() byte // Format should return one of the format constants defined by the proto package
	BytesPerSample() int
}

// Uint8Reader implements the Reader interface.
// The semantics are the same as io.Reader's Read.
type Uint8Reader func([]byte) (int, error)

// Int16Reader implements the Reader interface.
// The semantics are the same as io.Reader's Read, but it returns
// the number of int16 values read, not the number of bytes.
type Int16Reader func([]int16) (int, error)

// Int32Reader implements the Reader interface.
// The semantics are the same as io.Reader's Read, but it returns
// the number of int32 values read, not the number of bytes.
type Int32Reader func([]int32) (int, error)

// Float32Reader implements the Reader interface.
// The semantics are the same as io.Reader's Read, but it returns
// the number of float32 values read, not the number of bytes.
type Float32Reader func([]float32) (int, error)

func (c Uint8Reader) Read(buf []byte) (int, error) { return c(buf) }
func (c Uint8Reader) Format() byte                 { return proto.FormatUint8 }
func (c Uint8Reader) BytesPerSample() int          { return 1 }

func (c Int16Reader) Read(buf []byte) (int, error) {
	n, err := c(int16Slice(buf))
	return n * 2, err
}
func (c Int16Reader) Format() byte        { return formatI16 }
func (c Int16Reader) BytesPerSample() int { return 2 }

func (c Int32Reader) Read(buf []byte) (int, error) {
	n, err := c(int32Slice(buf))
	return n * 4, err
}
func (c Int32Reader) Format() byte        { return formatI32 }
func (c Int32Reader) BytesPerSample() int { return 2 }

func (c Float32Reader) Read(buf []byte) (int, error) {
	n, err := c(float32Slice(buf))
	return n * 4, err
}
func (c Float32Reader) Format() byte        { return formatF32 }
func (c Float32Reader) BytesPerSample() int { return 2 }

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
