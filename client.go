package pulse

import (
	"errors"
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

	server    string
	appName   string
	mediaName string
}

func NewClient(opts ...ClientOption) (*Client, error) {
	servers := []serverString{
		{
			protocol: "unix",
			addr:     fmt.Sprint("/run/user/", os.Getuid(), "/pulse/native"),
		},
	}

	c := &Client{
		appName:   path.Base(os.Args[0]),
		mediaName: "go audio",
	}
	for _, opt := range opts {
		opt(c)
	}

	if c.server != "" {
		servers = parseServerString(c.server)
	} else if serverRaw, ok := os.LookupEnv("PULSE_SERVER"); ok {
		servers = parseServerString(serverRaw)
	}
	if len(servers) == 0 {
		return nil, errors.New("no valid pulse server")
	}

	localname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, s := range servers {
		if s.localname != "" && localname != s.localname {
			continue
		}
		conn, err := net.Dial(s.protocol, s.addr)
		if err != nil {
			lastErr = err
			continue
		}

		c.conn = conn

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
				lastErr = err
				continue
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
			lastErr = err
			continue
		}
		c.c.SetVersion(authReply.Version)

		err = c.c.Request(&proto.SetClientName{Props: map[string]string{
			"media.name":                 c.mediaName,
			"application.name":           c.appName,
			"application.process.id":     fmt.Sprintf("%d", os.Getpid()),
			"application.process.binary": os.Args[0],
			"window.x11.display":         os.Getenv("DISPLAY"),
		}}, &proto.SetClientNameReply{})
		if err != nil {
			c.conn.Close()
			lastErr = err
			continue
		}

		return c, nil
	}
	return nil, fmt.Errorf("connections failed: %v", lastErr)
}

func (c *Client) Close() {
	c.conn.Close()
}

type ClientOption func(*Client)

func ClientApplicationName(name string) ClientOption {
	return func(c *Client) { c.appName = name }
}

func ClientMediaName(name string) ClientOption {
	return func(c *Client) { c.mediaName = name }
}

// see https://www.freedesktop.org/wiki/Software/PulseAudio/Documentation/User/ServerStrings/
func ClientServerString(s string) ClientOption {
	return func(c *Client) { c.server = s }
}
