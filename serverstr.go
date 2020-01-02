package pulse

import (
	"strings"
)

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
