package proto

import (
	"io"
	"reflect"
	"strconv"
)

type ProtocolWriter struct {
	w   io.Writer
	buf []byte
	pos int
	err error
}

func (p *ProtocolWriter) setErr(err error) {
	if p.err != nil {
		p.err = err
	}
}

func (p *ProtocolWriter) flush() {
	if p.err != nil {
		return
	}
	_, err := p.w.Write(p.buf[:p.pos])
	if err != nil {
		p.err = err
	}
	p.pos = 0
}

func (p *ProtocolWriter) ensure(n int) {
	if len(p.buf) < 1024 {
		p.flush()
		p.buf = make([]byte, 1024)
	}
	if len(p.buf)+p.pos < n {
		p.flush()
	}
}

func (p *ProtocolWriter) byte(b byte) {
	p.ensure(1)
	p.buf[p.pos] = b
	p.pos++
}

func (p *ProtocolWriter) uint32(u uint32) {
	p.ensure(4)
	p.buf[p.pos] = byte(u >> 24)
	p.buf[p.pos+1] = byte(u >> 16)
	p.buf[p.pos+2] = byte(u >> 8)
	p.buf[p.pos+3] = byte(u)
	p.pos += 4
}

func (p *ProtocolWriter) uint64(u uint64) {
	p.ensure(8)
	p.buf[p.pos] = byte(u >> 56)
	p.buf[p.pos+1] = byte(u >> 48)
	p.buf[p.pos+2] = byte(u >> 40)
	p.buf[p.pos+3] = byte(u >> 32)
	p.buf[p.pos+4] = byte(u >> 24)
	p.buf[p.pos+5] = byte(u >> 16)
	p.buf[p.pos+6] = byte(u >> 8)
	p.buf[p.pos+7] = byte(u)
	p.pos += 8
}

func (p *ProtocolWriter) string(s string) {
	p.ensure(len(s) + 1)
	copy(p.buf[p.pos:], s)
	p.buf[p.pos+len(s)] = 0
	p.pos += len(s) + 1
}

func (p *ProtocolWriter) x(x []byte) {
	p.uint32(uint32(len(x)))
	p.ensure(len(x))
	copy(p.buf[p.pos:], x)
	p.pos += len(x)
}

func (p *ProtocolWriter) xstring(x string) {
	p.uint32(uint32(len(x)) + 1)
	p.ensure(len(x))
	copy(p.buf[p.pos:], x)
	p.pos += len(x)
	p.byte(0)
}

func (p *ProtocolWriter) propList(list PropList) {
	for k, v := range list {
		p.byte('t')
		p.string(k)
		p.byte('L')
		p.uint32(uint32(len(v)))
		p.byte('x')
		p.x(v)
	}
	p.byte('N')
}

func (p *ProtocolWriter) value(i interface{}, version Version) {
	if i == nil {
		return
	}
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

		switch f := f.Interface().(type) {
		case string:
			if f == "" {
				p.byte('N')
			} else {
				p.byte('t')
				p.string(f)
			}
		case uint32:
			p.byte('L')
			p.uint32(f)
		case Version:
			p.byte('L')
			p.uint32(uint32(f))
		case byte:
			p.byte('B')
			p.byte(f)
		case uint64:
			p.byte('R')
			p.uint64(f)
		case int64:
			p.byte('r')
			p.uint64(uint64(f))
		case SampleSpec:
			p.byte('a')
			p.byte(f.Format)
			p.byte(f.Channels)
			p.uint32(f.Rate)
		case []byte:
			p.byte('x')
			p.x(f)
		case bool:
			if f {
				p.byte('1')
			} else {
				p.byte('0')
			}
		case Time:
			p.byte('T')
			p.uint32(f.Seconds)
			p.uint32(f.Microseconds)
		case Microseconds:
			p.byte('U')
			p.uint64(uint64(f))
		case ChannelMap:
			p.byte('m')
			p.byte(byte(len(f)))
			for i := range f {
				p.byte(f[i])
			}
		case ChannelVolumes:
			p.byte('v')
			p.byte(byte(len(f)))
			for i := range f {
				p.uint32(f[i])
			}
		case PropList:
			p.byte('P')
			p.propList(f)
		case Volume:
			p.byte('V')
			p.uint32(uint32(f))
		case FormatInfo:
			p.byte('f')
			p.byte('B')
			p.byte(f.Encoding)
			p.byte('P')
			p.propList(f.Properties)
		case []FormatInfo:
			p.byte('B')
			p.byte(byte(len(f)))
			for _, f := range f {
				p.byte('f')
				p.byte('B')
				p.byte(f.Encoding)
				p.byte('P')
				p.propList(f.Properties)
			}
		}
	}
}
