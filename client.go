package pulse

import (
	"fmt"
	"net"
	"os"
	"path"
	"sync"

	"github.com/jfreymuth/pulse/proto"
)

// The Client is the connection to the pulseaudio server. An application typically only uses a single client.
type Client struct {
	conn net.Conn
	c    *proto.Client

	mu       sync.Mutex
	playback map[uint32]*PlaybackStream
	record   map[uint32]*RecordStream

	server string
	props  proto.PropList
}

// NewClient connects to the server.
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		props: proto.PropList{
			"media.name":                 proto.PropListString("go audio"),
			"application.name":           proto.PropListString(path.Base(os.Args[0])),
			"application.icon_name":      proto.PropListString("audio-x-generic"),
			"application.process.id":     proto.PropListString(fmt.Sprintf("%d", os.Getpid())),
			"application.process.binary": proto.PropListString(os.Args[0]),
			"window.x11.display":         proto.PropListString(os.Getenv("DISPLAY")),
		},
	}
	for _, opt := range opts {
		opt(c)
	}

	var err error
	c.c, c.conn, err = proto.Connect(c.server)
	if err != nil {
		return nil, err
	}

	err = c.c.Request(&proto.SetClientName{Props: c.props}, &proto.SetClientNameReply{})
	if err != nil {
		c.conn.Close()
		return nil, err
	}

	c.playback = make(map[uint32]*PlaybackStream)
	c.record = make(map[uint32]*RecordStream)
	c.c.Callback = func(msg interface{}) {
		switch msg := msg.(type) {
		case *proto.Request:
			c.mu.Lock()
			stream, ok := c.playback[msg.StreamIndex]
			c.mu.Unlock()
			if ok {
				stream.request <- int(msg.Length)
			}
		case *proto.DataPacket:
			c.mu.Lock()
			stream, ok := c.record[msg.StreamIndex]
			c.mu.Unlock()
			if ok {
				stream.write(msg.Data)
			}
		case *proto.Started:
			c.mu.Lock()
			stream, ok := c.playback[msg.StreamIndex]
			c.mu.Unlock()
			if ok && stream.state == running && !stream.underflow {
				stream.started <- true
			}
		case *proto.Underflow:
			c.mu.Lock()
			stream, ok := c.playback[msg.StreamIndex]
			c.mu.Unlock()
			if ok {
				if stream.state == running {
					stream.underflow = true
				}
			}
		case *proto.ConnectionClosed:
			c.mu.Lock()
			for _, p := range c.playback {
				close(p.request)
				p.err = ErrConnectionClosed
				p.state = serverLost
			}
			for _, r := range c.record {
				r.err = ErrConnectionClosed
				r.state = serverLost
			}
			c.playback = make(map[uint32]*PlaybackStream)
			c.record = make(map[uint32]*RecordStream)
			c.mu.Unlock()
			c.conn.Close()
		default:
			//fmt.Printf("%#v\n", msg)
		}
	}

	return c, nil
}

// Close closes the client. Calling methods on a closed client may panic.
func (c *Client) Close() {
	c.conn.Close()
}

// A ClientOption supplies configuration when creating the client.
type ClientOption func(*Client)

// ClientApplicationName sets the application name.
// This will e.g. be displayed by a volume control application to identity the application.
// It should be human-readable and localized.
func ClientApplicationName(name string) ClientOption {
	return func(c *Client) { c.props["application.name"] = proto.PropListString(name) }
}

// ClientApplicationIconName sets the application icon using an xdg icon name.
// This will e.g. be displayed by a volume control application to identity the application.
func ClientApplicationIconName(name string) ClientOption {
	return func(c *Client) { c.props["application.icon_name"] = proto.PropListString(name) }
}

// ClientServerString will override the default server strings.
// Server strings are used to connect to the server. For the server string format see
// https://www.freedesktop.org/wiki/Software/PulseAudio/Documentation/User/ServerStrings/
func ClientServerString(s string) ClientOption {
	return func(c *Client) { c.server = s }
}

// RawRequest can be used to send arbitrary requests.
//
// req should be one of the request types defined by the proto package.
//
// rpl must be a pointer to the correct reply type or nil. This funcion will panic if rpl has the wrong type.
//
// The returned error can be compared against errors defined by the proto package to check for specific errors.
//
// The function will always block until the server has replied, even if rpl is nil.
func (c *Client) RawRequest(req proto.RequestArgs, rpl proto.Reply) error {
	return c.c.Request(req, rpl)
}

// ErrConnectionClosed is a special error value indicating that the server closed the connection.
const ErrConnectionClosed = pulseError("pulseaudio: connection closed")

type pulseError string

func (e pulseError) Error() string { return string(e) }
