package proto

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
)

type Client struct {
	r ProtocolReader
	w ProtocolWriter
	v Version

	replyM     sync.Mutex
	nextID     uint32
	awaitReply map[uint32]AwaitReply
	err        error

	send chan send

	Callback           func(interface{})
	OnConnectionClosed func()
}

type send struct {
	index uint32
	data  []byte
}

func (c *Client) Version() Version {
	return c.v
}

func (c *Client) SetVersion(v Version) {
	c.v = c.v.Min(v)
}

func (c *Client) Open(rw io.ReadWriter) {
	//debug, _ := os.Create("debug")
	//c.r.r = io.TeeReader(rw, debug)
	c.r.r = rw
	c.w.w = rw
	c.v = Version(32)

	c.send = make(chan send)
	c.awaitReply = make(map[uint32]AwaitReply)
	go c.readLoop()
	go c.writeLoop()
}

type AwaitReply struct {
	value interface{}
	reply chan<- error
}

func (c *Client) Request(req RequestArgs, rpl Reply) error {
	if rpl != nil && req.command() != rpl.IsReplyTo() {
		return fmt.Errorf("pulse: wrong reply type, got %d but expected %d", rpl.IsReplyTo(), req.command())
	}

	reply := make(chan error, 1)
	c.replyM.Lock()
	if c.err != nil {
		c.replyM.Unlock()
		return c.err
	}
	tag := c.nextID
	c.nextID++
	c.awaitReply[tag] = AwaitReply{rpl, reply}
	c.replyM.Unlock()
	defer func() {
		c.replyM.Lock()
		delete(c.awaitReply, tag)
		c.replyM.Unlock()
	}()

	var buf bytes.Buffer
	w := ProtocolWriter{w: &buf}
	w.byte('L')
	w.uint32(req.command())
	w.byte('L')
	w.uint32(tag)
	w.value(req, c.v)
	w.flush()

	c.send <- send{0xFFFFFFFF, buf.Bytes()}

	return <-reply
}

func (c *Client) Send(index uint32, data []byte) {
	c.send <- send{index, data}
}

func (c *Client) writeLoop() {
	for s := range c.send {
		c.w.uint32(uint32(len(s.data)))
		c.w.uint32(s.index)
		c.w.uint64(0)
		c.w.uint32(0)
		c.w.flush()
		c.w.w.Write(s.data)
	}
}

func (c *Client) readLoop() {
	for {
		length := c.r.uint32()
		index := c.r.uint32()
		offset := c.r.uint64()
		flags := c.r.uint32()
		_, _ = offset, flags
		if c.r.err != nil {
			c.error(c.err)
			return
		}
		if index == 0xFFFFFFFF {
			c.r.byte() // L
			op := c.r.uint32()
			c.r.byte() // L
			tag := c.r.uint32()
			var message interface{}
			switch op {
			case OpError:
				c.r.byte()
				err := Error(c.r.uint32())
				c.replyM.Lock()
				a, ok := c.awaitReply[tag]
				c.replyM.Unlock()
				if ok {
					a.reply <- err
				}
			case OpReply:
				c.replyM.Lock()
				a, ok := c.awaitReply[tag]
				c.replyM.Unlock()
				if ok {
					if a.value != nil {
						if reflect.TypeOf(a.value).Elem().Kind() == reflect.Slice {
							c.parseInfoList(a.value, int(length)-10)
						} else {
							c.r.value(a.value, c.v)
						}
					} else {
						c.r.advance(int(length) - 10)
					}
					a.reply <- nil
				} else {
					c.r.advance(int(length) - 10)
				}
			case OpRequest:
				message = &Request{}
			case OpOverflow:
				message = &Overflow{}
			case OpUnderflow:
				message = &Underflow{}
			case OpPlaybackStreamKilled:
				message = &PlaybackStreamKilled{}
			case OpRecordStreamKilled:
				message = &RecordStreamKilled{}
			case OpSubscribeEvent:
				message = &SubscribeEvent{}
			case OpPlaybackStreamSuspended:
				message = &PlaybackStreamSuspended{}
			case OpRecordStreamSuspended:
				message = &RecordStreamSuspended{}
			case OpPlaybackStreamMoved:
				message = &PlaybackStreamMoved{}
			case OpRecordStreamMoved:
				message = &RecordStreamMoved{}
			case OpClientEvent:
				message = &ClientEvent{}
			case OpPlaybackStreamEvent:
				message = &PlaybackStreamEvent{}
			case OpRecordStreamEvent:
				message = &RecordStreamEvent{}
			case OpStarted:
				message = &Started{}
			case OpPlaybackBufferAttrChanged:
				message = &PlaybackBufferAttrChanged{}
			default:
				fmt.Println(op)
				c.r.advance(int(length) - 10)
			}
			if message != nil {
				c.r.value(message, c.v)
				if c.Callback != nil {
					c.Callback(message)
				}
			}
		} else {
			if c.Callback != nil {
				buf := c.r.tmpbytes(int(length))
				c.Callback(&DataPacket{index, buf})
			}
			c.r.advance(int(length))
		}
	}
}

func (c *Client) error(err error) {
	c.replyM.Lock()
	c.err = err
	ch := make([]chan<- error, len(c.awaitReply))
	for _, a := range c.awaitReply {
		ch = append(ch, a.reply)
	}
	c.replyM.Unlock()
	for _, ch := range ch {
		ch <- err
	}
	if errors.Is(err, io.EOF) && (c.OnConnectionClosed != nil) {
		c.OnConnectionClosed()
	}
}

func (c *Client) parseInfoList(value interface{}, length int) {
	start := c.r.pos
	for c.r.pos-start < length {
		switch value := value.(type) {
		case *GetSinkInfoListReply:
			var v GetSinkInfoReply
			c.r.value(&v, c.v)
			*value = append(*value, &v)
		case *GetSourceInfoListReply:
			var v GetSourceInfoReply
			c.r.value(&v, c.v)
			*value = append(*value, &v)
		case *GetModuleInfoListReply:
			var v GetModuleInfoReply
			c.r.value(&v, c.v)
			*value = append(*value, &v)
		case *GetClientInfoListReply:
			var v GetClientInfoReply
			c.r.value(&v, c.v)
			*value = append(*value, &v)
		case *GetCardInfoListReply:
			var v GetCardInfoReply
			c.r.value(&v, c.v)
			*value = append(*value, &v)
		case *GetSinkInputInfoListReply:
			var v GetSinkInputInfoReply
			c.r.value(&v, c.v)
			*value = append(*value, &v)
		case *GetSourceOutputInfoListReply:
			var v GetSourceOutputInfoReply
			c.r.value(&v, c.v)
			*value = append(*value, &v)
		case *GetSampleInfoListReply:
			var v GetSampleInfoReply
			c.r.value(&v, c.v)
			*value = append(*value, &v)
		default:
			panic("wrong type")
		}
	}
}

type DataPacket struct {
	StreamIndex uint32
	Data        []byte
}
