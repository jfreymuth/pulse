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
}

// A Writer accepts audio data in a specific format.
type Writer interface {
	io.Writer
	Format() byte // Format should return one of the format constants defined by the proto package
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

// NewReader creates a reader from an io.Reader and a format.
// The format must be one of the constants defined in the proto package.
func NewReader(r io.Reader, format byte) Reader {
	check(format)
	return &reader{r, format}
}

// Uint8Writer implements the Writer interface.
// The semantics are the same as io.Writer's Write.
type Uint8Writer func([]byte) (int, error)

// Int16Writer implements the Writer interface.
// The semantics are the same as io.Writer's Write, but it returns
// the number of int16 values written, not the number of bytes.
type Int16Writer func([]int16) (int, error)

// Int32Writer implements the Writer interface.
// The semantics are the same as io.Writer's Write, but it returns
// the number of int32 values written, not the number of bytes.
type Int32Writer func([]int32) (int, error)

// Float32Writer implements the Writer interface.
// The semantics are the same as io.Writer's Write, but it returns
// the number of float32 values written, not the number of bytes.
type Float32Writer func([]float32) (int, error)

// NewWriter creates a writer from an io.Writer and a format.
// The format must be one of the constants defined in the proto package.
func NewWriter(w io.Writer, format byte) Writer {
	check(format)
	return &writer{w, format}
}

func (c Uint8Reader) Read(buf []byte) (int, error) { return c(buf) }
func (c Uint8Reader) Format() byte                 { return proto.FormatUint8 }

func (c Int16Reader) Read(buf []byte) (int, error) {
	n, err := c(int16Slice(buf))
	return n * 2, err
}
func (c Int16Reader) Format() byte { return formatI16 }

func (c Int32Reader) Read(buf []byte) (int, error) {
	n, err := c(int32Slice(buf))
	return n * 4, err
}
func (c Int32Reader) Format() byte { return formatI32 }

func (c Float32Reader) Read(buf []byte) (int, error) {
	n, err := c(float32Slice(buf))
	return n * 4, err
}
func (c Float32Reader) Format() byte { return formatF32 }

func (c Uint8Writer) Write(buf []byte) (int, error) { return c(buf) }
func (c Uint8Writer) Format() byte                  { return proto.FormatUint8 }

func (c Int16Writer) Write(buf []byte) (int, error) {
	n, err := c(int16Slice(buf))
	return n * 2, err
}
func (c Int16Writer) Format() byte { return formatI16 }

func (c Int32Writer) Write(buf []byte) (int, error) {
	n, err := c(int32Slice(buf))
	return n * 4, err
}
func (c Int32Writer) Format() byte { return formatI32 }

func (c Float32Writer) Write(buf []byte) (int, error) {
	n, err := c(float32Slice(buf))
	return n * 4, err
}
func (c Float32Writer) Format() byte { return formatF32 }

type reader struct {
	r io.Reader
	f byte
}

func (r *reader) Read(buf []byte) (int, error) { return r.r.Read(buf) }
func (r *reader) Format() byte                 { return r.f }

type writer struct {
	w io.Writer
	f byte
}

func (w *writer) Write(buf []byte) (int, error) { return w.w.Write(buf) }
func (w *writer) Format() byte                  { return w.f }

func bytes(f byte) int {
	switch f {
	case proto.FormatUint8:
		return 1
	case proto.FormatInt16LE, proto.FormatInt16BE:
		return 2
	case proto.FormatInt32LE, proto.FormatInt32BE, proto.FormatFloat32LE, proto.FormatFloat32BE:
		return 4
	}
	panic("pulse: invalid format")
}

func check(f byte) {
	switch f {
	case proto.FormatUint8, proto.FormatInt16LE, proto.FormatInt16BE,
		proto.FormatInt32LE, proto.FormatInt32BE, proto.FormatFloat32LE, proto.FormatFloat32BE:
		return
	}
	panic("pulse: invalid format")
}

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
