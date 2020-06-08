package proto

import (
	"io"
	"reflect"
	"strconv"
)

type ProtocolReader struct {
	r        io.Reader
	buf      []byte
	buffered int
	err      error
	pos      int
}

func (p *ProtocolReader) setErr(err error) {
	if p.err != nil {
		p.err = err
	}
}

func (p *ProtocolReader) fill(min int) {
	const bufferSize = 1024
	if p.err != nil {
		return
	}
	if len(p.buf) < min {
		size := 2 * min
		if size < bufferSize {
			size = bufferSize
		}
		buf := make([]byte, size)
		copy(buf, p.buf[:p.buffered])
		p.buf = buf
	}
	emptyReads := 0
	for p.buffered < min {
		n, err := p.r.Read(p.buf[p.buffered:])
		p.buffered += n
		if err != nil {
			p.err = err
			return
		}
		if n == 0 {
			emptyReads++
			if emptyReads >= 100 {
				p.err = io.ErrNoProgress
				return
			}
		}
	}
}

func (p *ProtocolReader) advance(n int) {
	p.fill(n)
	if p.err != nil {
		return
	}
	p.buf = p.buf[n:]
	p.buffered -= n
	p.pos += n
}

func (p *ProtocolReader) byte() byte {
	p.fill(1)
	if p.err != nil {
		return 0
	}
	b := p.buf[0]
	p.advance(1)
	return b
}

func (p *ProtocolReader) uint32() uint32 {
	p.fill(4)
	if p.err != nil {
		return 0
	}
	u := uint32(p.buf[0])<<24 | uint32(p.buf[1])<<16 | uint32(p.buf[2])<<8 | uint32(p.buf[3])
	p.advance(4)
	return u
}

func (p *ProtocolReader) uint64() uint64 {
	p.fill(8)
	if p.err != nil {
		return 0
	}
	u := uint64(p.buf[0])<<56 | uint64(p.buf[1])<<48 | uint64(p.buf[2])<<40 | uint64(p.buf[3])<<32 | uint64(p.buf[4])<<24 | uint64(p.buf[5])<<16 | uint64(p.buf[6])<<8 | uint64(p.buf[7])
	p.advance(8)
	return u
}

func (p *ProtocolReader) bool() bool {
	return p.byte() == '1'
}

func (p *ProtocolReader) string() string {
	const maxLength = 1024
	for i := 0; i < maxLength; i++ {
		if i >= p.buffered {
			p.fill(p.buffered + 1)
		}
		if p.err != nil {
			return ""
		}
		if p.buf[i] == 0 {
			s := string(p.buf[:i])
			p.advance(i + 1)
			return s
		}
	}
	p.setErr(ErrProtocolError)
	return ""
}

func (p *ProtocolReader) bytes(out []byte) {
	p.fill(len(out))
	if p.err != nil {
		return
	}
	copy(out, p.buf)
	p.advance(len(out))
}

func (p *ProtocolReader) tmpbytes(n int) []byte {
	p.fill(n)
	if p.err != nil {
		return nil
	}
	return p.buf[:n]
}

func (p *ProtocolReader) x() []byte {
	l := p.uint32()
	x := make([]byte, l)
	p.fill(int(l))
	if p.err != nil {
		return nil
	}
	copy(x, p.buf)
	p.advance(int(l))
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
