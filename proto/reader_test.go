package proto

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestProtocolReaderUint32(t *testing.T) {
	r := ProtocolReader{
		r: bufio.NewReader(prepareUint32Buf()),
	}

	for i := uint32(0); i < 1000; i++ {
		d := r.uint32()
		if r.err != nil {
			t.Errorf("expecting no error, got %v", r.err)
			return
		}

		if d != i {
			t.Errorf("expecting read %d, got %d", i, d)
			return
		}
	}

	if r.pos != 4000 {
		t.Errorf("expecting final pos %d, got %d", 4000, r.pos)
		return
	}
}

func BenchmarkProtocolReaderUint32(b *testing.B) {
	buf := prepareUint32Buf().Bytes()

	var r ProtocolReader

	reader := bufio.NewReader(bytes.NewReader(buf))
	r.r = reader

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		reader.Reset(bytes.NewReader(buf))
		for i := uint32(0); i < 1000; i++ {
			d := r.uint32()
			if r.err != nil {
				b.Errorf("expecting no error, got %v", r.err)
				return
			}

			if d != i {
				b.Errorf("expecting read %d, got %d", i, d)
				return
			}
		}
	}
}

func prepareUint32Buf() *bytes.Buffer {
	var buf bytes.Buffer

	for i := uint32(0); i < 1000; i++ {
		err := binary.Write(&buf, binary.BigEndian, i)
		if err != nil {
			panic(err)
		}
	}

	return &buf
}

func prepareUint64Buf() *bytes.Buffer {
	var buf bytes.Buffer

	for i := uint64(0); i < 1000; i++ {
		err := binary.Write(&buf, binary.BigEndian, i)
		if err != nil {
			panic(err)
		}
	}

	return &buf
}

func TestProtocolReaderUint64(t *testing.T) {
	r := ProtocolReader{
		r: bufio.NewReader(prepareUint64Buf()),
	}

	for i := uint64(0); i < 1000; i++ {
		d := r.uint64()
		if r.err != nil {
			t.Errorf("expecting no error, got %v", r.err)
			return
		}

		if d != i {
			t.Errorf("expecting read %d, got %d", i, d)
			return
		}
	}

	if r.pos != 8000 {
		t.Errorf("expecting final pos %d, got %d", 8000, r.pos)
		return
	}
}

func prepareByteBuf() *bytes.Buffer {
	var buf bytes.Buffer

	for i := byte(0); i < 255; i++ {
		err := binary.Write(&buf, binary.BigEndian, i)
		if err != nil {
			panic(err)
		}
	}

	return &buf
}

func TestProtocolReaderByte(t *testing.T) {
	r := ProtocolReader{
		r: bufio.NewReader(prepareByteBuf()),
	}

	for i := byte(0); i < 255; i++ {
		d := r.byte()
		if r.err != nil {
			t.Errorf("expecting no error, got %v", r.err)
			return
		}

		if d != i {
			t.Errorf("expecting read %d, got %d", i, d)
			return
		}
	}

	if r.pos != 255 {
		t.Errorf("expecting final pos %d, got %d", 255, r.pos)
		return
	}
}

func TestProtocolReaderBytes(t *testing.T) {
	r := ProtocolReader{
		r: bufio.NewReader(prepareUint32Buf()),
	}

	b := make([]byte, 4000)

	r.bytes(b)

	for i := uint32(0); i < 1000; i++ {
		u := binary.BigEndian.Uint32(b[i*4 : (i+1)*4])

		if u != i {
			t.Errorf("expecting %d, got %d", i, u)
			return
		}
	}

	if r.pos != 4000 {
		t.Errorf("expecting final pos %d, got %d", 4000, r.pos)
		return
	}
}

func TestProtocolReaderTmpBytes(t *testing.T) {
	r := ProtocolReader{
		r: bufio.NewReader(prepareUint32Buf()),
	}

	b := r.tmpbytes(4000)

	for i := uint32(0); i < 1000; i++ {
		u := binary.BigEndian.Uint32(b[i*4 : (i+1)*4])

		if u != i {
			t.Errorf("expecting %d, got %d", i, u)
			return
		}
	}

	if r.pos != 0 {
		t.Errorf("expecting final pos %d, got %d", 0, r.pos)
		return
	}
}

func TestProtocolReaderString(t *testing.T) {
	original := "Lorem ipsum dolor sit amet, consectetur adipiscing elit.\x00"
	r := ProtocolReader{
		r: bufio.NewReader(strings.NewReader(original)),
	}

	s := r.string()

	if s != original[:len(original)-1] {
		t.Errorf("expecting %s, got %s", s, original[:len(original)-1])
		return
	}

	if r.pos != len(original) {
		t.Errorf("expecting final pos %d, got %d", len(original), r.pos)
		return
	}
}

func prepareXBuf() *bytes.Buffer {
	var buf bytes.Buffer

	err := binary.Write(&buf, binary.BigEndian, uint32(1000))
	if err != nil {
		panic(err)
	}

	for i := uint32(0); i < 250; i++ {
		err := binary.Write(&buf, binary.BigEndian, i)
		if err != nil {
			panic(err)
		}
	}

	return &buf
}

func TestProtocolReaderX(t *testing.T) {
	r := ProtocolReader{
		r: bufio.NewReader(prepareXBuf()),
	}

	b := r.x()

	if len(b) != 1000 {
		t.Errorf("expecting length of %d, got %d", 1000, len(b))
		return
	}

	for i := uint32(0); i < 250; i++ {
		u := binary.BigEndian.Uint32(b[i*4 : (i+1)*4])

		if u != i {
			t.Errorf("expecting %d, got %d", i, u)
			return
		}
	}

	if r.pos != 1004 {
		t.Errorf("expecting final pos %d, got %d", 1004, r.pos)
		return
	}
}
