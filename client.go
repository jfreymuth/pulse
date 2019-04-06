package pulse

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"sync"

	"github.com/jfreymuth/pulse/proto"
)

type Client struct {
	conn net.Conn
	c    proto.Client

	mu       sync.Mutex
	playback map[uint32]*PlaybackStream
	record   map[uint32]*RecordStream
}

type PlaybackStream struct {
	index uint32
	buf   []byte
	cb    func([]byte)
}

type RecordStream struct {
	index uint32
	cb    func([]byte)
}

func NewClient() (*Client, error) {
	conn, err := net.Dial("unix", fmt.Sprint("/run/user/", os.Getuid(), "/pulse/native"))
	if err != nil {
		return nil, err
	}

	c := &Client{conn: conn}
	c.playback = make(map[uint32]*PlaybackStream)
	c.record = make(map[uint32]*RecordStream)
	c.c.Callback = func(msg interface{}) {
		switch msg := msg.(type) {
		case *proto.Request:
			c.mu.Lock()
			if stream, ok := c.playback[msg.StreamIndex]; ok {
				c.mu.Unlock()
				if len(stream.buf) < int(msg.Length) {
					stream.buf = make([]byte, msg.Length)
				}
				stream.cb(stream.buf[:msg.Length])
				c.c.Send(msg.StreamIndex, stream.buf[:msg.Length])
			} else {
				c.mu.Unlock()
			}
		case *proto.DataPacket:
			c.mu.Lock()
			if stream, ok := c.record[msg.StreamIndex]; ok {
				c.mu.Unlock()
				stream.cb(msg.Data)
			} else {
				c.mu.Unlock()
			}
		default:
			//fmt.Printf("%#v\n", msg)
		}
	}
	c.c.Open(conn)

	cookie, err := ioutil.ReadFile(os.Getenv("HOME") + "/.config/pulse/cookie")
	if err != nil {
		c.conn.Close()
		return nil, err
	}
	var authReply proto.AuthReply
	err = c.c.Request(&proto.Auth{Version: c.c.Version(), Cookie: cookie}, &authReply)
	if err != nil {
		c.conn.Close()
		return nil, err
	}
	c.c.SetVersion(authReply.Version)

	err = c.c.Request(&proto.SetClientName{map[string]string{
		"media.name":                 "go audio",
		"application.name":           path.Base(os.Args[0]),
		"application.process.id":     fmt.Sprintf("%d", os.Getpid()),
		"application.process.binary": os.Args[0],
		"window.x11.display":         os.Getenv("DISPLAY"),
	}}, &proto.SetClientNameReply{})
	if err != nil {
		c.conn.Close()
		return nil, err
	}

	return c, nil
}

func (c *Client) CreatePlayback(rate int, cb func([]byte)) (*PlaybackStream, error) {
	var reply proto.CreatePlaybackStreamReply
	err := c.c.Request(&proto.CreatePlaybackStream{
		SampleSpec:            proto.SampleSpec{Format: proto.FormatFloat32LE, Channels: 1, Rate: uint32(rate)},
		ChannelMap:            []byte{0},
		SinkIndex:             0xFFFFFFFF,
		BufferMaxLength:       0xFFFFFFFF,
		BufferTargetLength:    0xFFFFFFFF,
		BufferPrebufferLength: 0xFFFFFFFF,
		BufferMinimumRequest:  0xFFFFFFFF,
		ChannelVolumes:        []uint32{256},
	}, &reply)
	if err != nil {
		return nil, err
	}
	stream := &PlaybackStream{
		index: reply.StreamIndex,
		cb:    cb,
	}
	c.mu.Lock()
	c.playback[stream.index] = stream
	c.mu.Unlock()
	c.c.Callback(&proto.Request{stream.index, reply.Missing})
	return stream, nil
}

func (c *Client) CreateRecord(rate int, cb func([]byte)) (*RecordStream, error) {
	var reply proto.CreateRecordStreamReply
	err := c.c.Request(&proto.CreateRecordStream{
		SampleSpec:         proto.SampleSpec{Format: proto.FormatFloat32LE, Channels: 1, Rate: uint32(rate)},
		ChannelMap:         []byte{0},
		SourceIndex:        0xFFFFFFFF,
		BufferMaxLength:    0xFFFFFFFF,
		BufferFragSize:     0xFFFFFFFF,
		ChannelVolumes:     []uint32{256},
		DirectOnInputIndex: 0xFFFFFFFF,
	}, &reply)
	if err != nil {
		return nil, err
	}
	stream := &RecordStream{
		index: reply.StreamIndex,
		cb:    cb,
	}
	c.mu.Lock()
	c.record[stream.index] = stream
	c.mu.Unlock()
	return stream, nil
}

func (c *Client) Close() {
	c.conn.Close()
}
