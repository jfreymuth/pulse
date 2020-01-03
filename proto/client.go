package proto

import (
	"bytes"
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

	err     chan error
	request chan uint32

	Callback func(interface{})
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

	c.awaitReply = make(map[uint32]AwaitReply)
	c.err = make(chan error, 4)
	c.request = make(chan uint32, 4)
	go c.readLoop()
}

type AwaitReply struct {
	value interface{}
	reply chan<- interface{}
}

func (c *Client) Request(req RequestArgs, rpl Reply) error {
	if rpl != nil && req.command() != rpl.IsReplyTo() {
		panic("pulse: wrong reply type")
	}

	c.replyM.Lock()
	tag := c.nextID
	c.nextID++
	reply := make(chan interface{}, 1)
	c.awaitReply[tag] = AwaitReply{rpl, reply}
	c.replyM.Unlock()

	var buf bytes.Buffer
	w := ProtocolWriter{w: &buf}
	w.byte('L')
	w.uint32(req.command())
	w.byte('L')
	w.uint32(tag)
	w.value(req, c.v)
	w.flush()

	c.Send(0xFFFFFFFF, buf.Bytes())

	select {
	case <-reply:
		return nil
	case err := <-c.err:
		return err
	}
}

func (c *Client) Send(index uint32, data []byte) {
	c.w.uint32(uint32(len(data)))
	c.w.uint32(index)
	c.w.uint64(0)
	c.w.uint32(0)
	c.w.flush()
	c.w.w.Write(data)
}

func (c *Client) readLoop() {
	for {
		length := c.r.uint32()
		index := c.r.uint32()
		offset := c.r.uint64()
		flags := c.r.uint32()
		_, _ = offset, flags
		if c.r.err != nil {
			for {
				c.err <- c.r.err
			}
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
				err := fmt.Errorf("pulse: %s", errorStrings[c.r.uint32()])
				for {
					c.err <- err
				}
			case OpReply:
				if a, ok := c.awaitReply[tag]; ok {
					if a.value != nil {
						if reflect.TypeOf(a.value).Elem().Kind() == reflect.Slice {
							c.parseInfoList(a.value, int(length)-10)
						} else {
							c.r.value(a.value, c.v)
						}
					} else {
						c.r.advance(int(length) - 10)
					}
					a.reply <- a.value
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

var errorStrings = []string{
	"ok",
	"access denied",
	"unknown command",
	"invalid argument",
	"entity exists",
	"no such entity",
	"connection refused",
	"protocol error",
	"timeout",
	"no authentication key",
	"internal error",
	"connection terminated",
	"entity killed",
	"invalid server",
	"module initialization failed",
	"bad state",
	"no data",
	"incompatible protocol version",
	"too large",
	"not supported",
	"unknown error code",
	"no such extension",
	"obsolete functionality",
	"missing implementation",
	"client forked",
	"input/output error",
	"device or resource busy",
}
