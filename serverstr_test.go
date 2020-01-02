package pulse

import (
	"reflect"
	"testing"
)

func TestParseServerString(t *testing.T) {
	cases := []struct {
		input  string
		result []serverString
	}{
		{
			"/path/to/socket",
			[]serverString{
				{"", "unix", "/path/to/socket"},
			},
		},
		{
			"tcp4:host:port",
			[]serverString{
				{"", "tcp4", "host:port"},
			},
		},
		{
			"tcp6:host:port",
			[]serverString{
				{"", "tcp6", "host:port"},
			},
		},
		{
			"tcp:address:port",
			[]serverString{
				{"", "tcp", "address:port"},
			},
		},
		{
			"{somewhere}/path/to/socket tcp:address:port",
			[]serverString{
				{"somewhere", "unix", "/path/to/socket"},
				{"", "tcp", "address:port"},
			},
		},
	}
	for _, c := range cases {
		s := parseServerString(c.input)
		if !reflect.DeepEqual(c.result, s) {
			t.Errorf("Expected parse result: %+v, but got: %+v", c.result, s)
		}
	}
}
