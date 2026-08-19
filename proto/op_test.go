package proto

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"
)

func TestSendObjectMessageEncoding(t *testing.T) {
	var buf bytes.Buffer
	w := ProtocolWriter{w: &buf}
	w.value(&SendObjectMessage{ObjectPath: "/core", Message: "list-handlers"}, Version(35))
	w.flush()

	want := []byte("t/core\x00tlist-handlers\x00N")
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("encoded %q, want %q", buf.Bytes(), want)
	}
}

func TestSendObjectMessageReplyDecoding(t *testing.T) {
	r := ProtocolReader{r: bufio.NewReader(bytes.NewReader([]byte("t[]\x00")))}
	var rpl SendObjectMessageReply
	r.value(&rpl, Version(35))
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if rpl.Response != "[]" {
		t.Errorf("Response = %q, want %q", rpl.Response, "[]")
	}
}

func TestSendObjectMessageRequiresVersion35(t *testing.T) {
	c := &Client{v: Version(34)}
	err := c.Request(&SendObjectMessage{ObjectPath: "/core", Message: "list-handlers"}, nil)
	if err != ErrNotSupported {
		t.Errorf("Request() error = %v, want %v", err, ErrNotSupported)
	}
}

// Port availability group and type were added in protocol version 34 and
// must only be consumed when negotiated.
func TestPortVersion34Fields(t *testing.T) {
	v33 := []byte("tline\x00tLine Out\x00L\x00\x00\x00\x09L\x00\x00\x00\x02")
	v34 := append(append([]byte{}, v33...), []byte("tLegacy 4\x00L\x00\x00\x00\x04")...)

	tests := []struct {
		name    string
		version Version
		data    []byte
		group   string
		typ     uint32
	}{
		{"v33 leaves new fields zero", 33, v33, "", 0},
		{"v34 decodes new fields", 34, v34, "Legacy 4", 4},
	}
	for _, reply := range []interface{}{GetSinkInfoReply{}, GetSourceInfoReply{}} {
		ports, _ := reflect.TypeOf(reply).FieldByName("Ports")
		for _, tt := range tests {
			t.Run(reflect.TypeOf(reply).Name()+"/"+tt.name, func(t *testing.T) {
				r := ProtocolReader{r: bufio.NewReader(bytes.NewReader(tt.data))}
				port := reflect.New(ports.Type.Elem())
				r.value(port.Interface(), tt.version)
				if r.err != nil {
					t.Fatalf("unexpected error: %v", r.err)
				}
				p := port.Elem()
				if p.FieldByName("Name").String() != "line" || p.FieldByName("Available").Uint() != 2 {
					t.Errorf("base fields = %+v", p.Interface())
				}
				group, typ := p.FieldByName("AvailabilityGroup").String(), uint32(p.FieldByName("Type").Uint())
				if group != tt.group || typ != tt.typ {
					t.Errorf("v34 fields = %q/%d, want %q/%d", group, typ, tt.group, tt.typ)
				}
				if r.pos != len(tt.data) {
					t.Errorf("consumed %d bytes, want %d", r.pos, len(tt.data))
				}
			})
		}
	}
}
