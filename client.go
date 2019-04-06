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
			stream, ok := c.playback[msg.StreamIndex]
			c.mu.Unlock()
			if ok {
				c.c.Send(msg.StreamIndex, stream.buffer(int(msg.Length)))
			}
		case *proto.DataPacket:
			c.mu.Lock()
			stream, ok := c.record[msg.StreamIndex]
			c.mu.Unlock()
			if ok {
				stream.write(msg.Data)
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

func (c *Client) Close() {
	c.conn.Close()
}
