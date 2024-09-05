package proto

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"strconv"
)

type ProtocolReader struct {
	r      *bufio.Reader
	err    error
	pos    int
	intBuf [8]byte
	buf    bytes.Buffer
}

func (p *ProtocolReader) setErr(err error) {
	if p.err != nil {
		p.err = err
	}
}

func (p *ProtocolReader) advance(n int) {
	p.tmpbytes(n)
	p.pos += n
}

func (p *ProtocolReader) byte() byte {
	_, err := io.ReadFull(p.r, p.intBuf[:1])
	if err != nil {
		p.err = err
		return 0
	}
	p.pos += 1
	return p.intBuf[0]
}

func (p *ProtocolReader) uint32() uint32 {
	_, err := io.ReadFull(p.r, p.intBuf[:4])
	if err != nil {
		p.err = err
		return 0
	}
	p.pos += 4
	return binary.BigEndian.Uint32(p.intBuf[:4])
}

func (p *ProtocolReader) uint64() uint64 {
	_, err := io.ReadFull(p.r, p.intBuf[:8])
	if err != nil {
		p.err = err
		return 0
	}
	p.pos += 8
	return binary.BigEndian.Uint64(p.intBuf[:8])
}

func (p *ProtocolReader) bool() bool {
	return p.byte() == '1'
}

func (p *ProtocolReader) string() string {
	const maxLength = 1024

	p.buf.Reset()

	// Modified from bufio.Reader.ReadBytes to use our own buffer
	// without allocating
	// Use ReadSlice to look for delim, accumulating full buffers.
	for {
		frag, err := p.r.ReadSlice(0)
		if err == nil { // got final fragment
			p.buf.Write(frag)
			break
		}

		if err != bufio.ErrBufferFull { // unexpected error
			p.err = err
			return ""
		}

		p.buf.Write(frag)

		if p.buf.Len() > 1024 {
			p.setErr(ErrProtocolError)
			return ""
		}
	}

	if p.buf.Len() == 0 {
		return ""
	}

	p.pos += p.buf.Len()

	return string(p.buf.Bytes()[:p.buf.Len()-1])
}

func (p *ProtocolReader) bytes(out []byte) {
	_, err := io.ReadFull(p.r, out)
	if err != nil {
		p.err = err
		return
	}

	p.pos += len(out)
}

func (p *ProtocolReader) tmpbytes(n int) []byte {
	p.buf.Reset()

	_, err := io.CopyN(&p.buf, p.r, int64(n))
	if err != nil {
		p.err = err
		return nil
	}
	return p.buf.Bytes()
}

func (p *ProtocolReader) x() []byte {
	l := p.uint32()
	x := make([]byte, l)
	p.bytes(x)
	if p.err != nil {
		return nil
	}
	return x
}

func (p *ProtocolReader) propList(out PropList) {
	for p.err == nil {
		keyType := p.byte()
		if keyType == 'N' {
			break
		}
		if keyType != 't' {
			p.setErr(ErrProtocolError)
			return
		}
		key := p.string()
		lenType := p.byte()
		if lenType != 'L' {
			p.setErr(ErrProtocolError)
			return
		}
		l := p.uint32()
		valueType := p.byte()
		if valueType != 'x' {
			p.setErr(ErrProtocolError)
			return
		}
		value := p.x()
		if len(value) != int(l) {
			p.setErr(ErrProtocolError)
			return
		}
		out[key] = PropListEntry(value)
	}
}

func (p *ProtocolReader) value(i interface{}, version Version) {
	v := reflect.ValueOf(i).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if p.err != nil {
			return
		}

		f := v.Field(i)

		if tag := string(t.Field(i).Tag); tag != "" {
			if ver, err := strconv.Atoi(tag); err == nil && ver > version.Version() {
				continue
			}
			if tag[0] == '<' {
				if ver, err := strconv.Atoi(tag[1:]); err == nil && ver <= version.Version() {
					continue
				}
			}
		}

		if f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.Struct {
			if _, ok := f.Interface().([]FormatInfo); ok {
				p.byte() // B
				l := p.byte()
				fi := make([]FormatInfo, l)
				for i := range fi {
					p.byte() // f
					p.byte() // B
					fi[i].Encoding = p.byte()
					p.byte() // P
					fi[i].Properties = make(PropList)
					p.propList(fi[i].Properties)
				}
			} else {
				p.byte() // L
				l := int(p.uint32())
				fv := reflect.MakeSlice(f.Type(), l, l)
				for i := 0; i < l; i++ {
					p.value(fv.Index(i).Addr().Interface(), version)
				}
				f.Set(fv)
			}
			continue
		}
		typ := p.byte()
		switch typ {
		case 't':
			f.SetString(p.string())
		case 'N':
			f.SetString("")
		case 'L':
			f.SetUint(uint64(p.uint32()))
		case 'B':
			f.SetUint(uint64(p.byte()))
		case 'R':
			f.SetUint(p.uint64())
		case 'r':
			f.SetInt(int64(p.uint64()))
		case 'a':
			f.Set(reflect.ValueOf(SampleSpec{p.byte(), p.byte(), p.uint32()}))
		case 'x':
			if f.Kind() == reflect.String {
				x := p.x()
				f.SetString(string(x[:len(x)-1]))
			} else {
				f.SetBytes(p.x())
			}
		case '1':
			f.SetBool(true)
		case '0':
			f.SetBool(false)
		case 'T':
			f.Set(reflect.ValueOf(Time{p.uint32(), p.uint32()}))
		case 'U':
			f.SetUint(p.uint64())
		case 'm':
			b := make([]byte, p.byte())
			p.bytes(b)
			f.SetBytes(b)
		case 'v':
			u := make([]uint32, p.byte())
			for i := range u {
				u[i] = p.uint32()
			}
			f.Set(reflect.ValueOf(u))
		case 'P':
			m := make(PropList)
			p.propList(m)
			f.Set(reflect.ValueOf(m))
		case 'V':
			f.SetUint(uint64(p.uint32()))
		case 'f':
			m := make(PropList)
			p.byte() // B
			enc := p.byte()
			p.byte() // P
			p.propList(m)
			f.Set(reflect.ValueOf(FormatInfo{enc, m}))
		}
	}
}
