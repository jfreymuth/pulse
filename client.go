package pulse

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
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
	socketNetwork := "unix"
	socketPath := fmt.Sprint("/run/user/", os.Getuid(), "/pulse/native")
	if pathRaw, ok := os.LookupEnv("PULSE_SERVER"); ok {
		path := strings.SplitN(pathRaw, ":", 2)
		switch len(path) {
		case 1:
			// If the network type is not specified, it should be handled as unix socket.
			socketPath = path[0]
		case 2:
			socketNetwork = path[0]
			socketPath = path[1]
		default:
			return nil, errors.New("invalid PULSE_SERVER string")
		}
	}

	conn, err := net.Dial(socketNetwork, socketPath)
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

	cookiePath := os.Getenv("HOME") + "/.config/pulse/cookie"
	if path, ok := os.LookupEnv("PULSE_COOKIE"); ok {
		cookiePath = path
	}

	cookie, err := ioutil.ReadFile(cookiePath)
	if err != nil {
		if !os.IsNotExist(err) {
			c.conn.Close()
			return nil, err
		}
		// If the server is launched with auth-anonymous=1,
		// any 256 bytes cookie will be accepted.
		cookie = make([]byte, 256)
	}
	var authReply proto.AuthReply
	err = c.c.Request(
		&proto.Auth{
			Version: c.c.Version(),
			Cookie:  cookie,
		}, &authReply)
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
