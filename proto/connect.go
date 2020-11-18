package proto

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"runtime"
	"strings"
)

// Connect connects to the pulse server.
//
// For the server string format see
// https://www.freedesktop.org/wiki/Software/PulseAudio/Documentation/User/ServerStrings/
// If the server string is empty, the environment variable PULSE_SERVER will be used.
func Connect(server string) (*Client, net.Conn, error) {
	sstr := []serverString{
		{
			protocol: "unix",
			addr:     addr(),
		},
	}
	if server != "" {
		sstr = parseServerString(server)
	} else if serverRaw, ok := os.LookupEnv("PULSE_SERVER"); ok {
		sstr = parseServerString(serverRaw)
	}
	if len(sstr) == 0 {
		return nil, nil, errors.New("pulseaudio: no valid server")
	}
	c := &Client{}

	localname, err := os.Hostname()
	if err != nil {
		return nil, nil, err
	}

	var lastErr error
	for _, s := range sstr {
		if s.localname != "" && localname != s.localname {
			continue
		}
		conn, err := net.Dial(s.protocol, s.addr)
		if err != nil {
			lastErr = err
			continue
		}
		c.Open(conn)

		cookiePath := os.Getenv("HOME") + "/.config/pulse/cookie"
		if path, ok := os.LookupEnv("PULSE_COOKIE"); ok {
			cookiePath = path
		}

		cookie, err := ioutil.ReadFile(cookiePath)
		if err != nil {
			if !os.IsNotExist(err) {
				conn.Close()
				lastErr = err
				continue
			}
			// If the server is launched with auth-anonymous=1,
			// any 256 bytes cookie will be accepted.
			cookie = make([]byte, 256)
		}
		var authReply AuthReply
		err = c.Request(
			&Auth{
				Version: c.Version(),
				Cookie:  cookie,
			}, &authReply)
		if err != nil {
			conn.Close()
			lastErr = err
			continue
		}
		c.SetVersion(authReply.Version)

		return c, conn, nil
	}

	return nil, nil, lastErr
}

type serverString struct {
	localname string
	protocol  string
	addr      string
}

func parseServerString(str string) []serverString {
	s := strings.Fields(str)
	var result []serverString
	for _, s := range s {
		var server serverString
		if s[0] == '{' {
			end := strings.IndexByte(s, '}')
			server.localname = s[1:end]
			s = s[end+1:]
		}
		switch {
		case len(s) == 0:
			// no server string
			continue
		case s[0] == '/':
			server.protocol = "unix"
			server.addr = s
		case strings.HasPrefix(s, "unix:"):
			server.protocol = "unix"
			server.addr = s[5:]
		case strings.HasPrefix(s, "tcp6:"):
			server.protocol = "tcp6"
			server.addr = s[5:]
		case strings.HasPrefix(s, "tcp4:"):
			server.protocol = "tcp4"
			server.addr = s[5:]
		case strings.HasPrefix(s, "tcp:"):
			server.protocol = "tcp"
			server.addr = s[4:]
		default:
			// invalid server string
			continue
		}
		result = append(result, server)
	}
	return result
}

func addr() (addr string) {
	switch runtime.GOOS {
	case "linux":
		addr = fmt.Sprint("/run/user/", os.Getuid(), "/pulse/native")
	case "darwin":
		u, err := user.Current()
		if err != nil {
			return
		}

		h, err := os.Hostname()
		if err != nil {
			return
		}

		addr = fmt.Sprintf("%s/.config/pulse/%s-runtime/native", u.HomeDir, h)
	}

	return
}
