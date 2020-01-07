package pulse

import "github.com/jfreymuth/pulse/proto"

// A Source is an input device.
type Source struct {
	info proto.GetSourceInfoReply
}

// ListSources returns a list of all available input devices.
func (c *Client) ListSources() ([]*Source, error) {
	var reply proto.GetSourceInfoListReply
	err := c.c.Request(&proto.GetSourceInfoList{}, &reply)
	if err != nil {
		return nil, err
	}
	sinks := make([]*Source, len(reply))
	for i := range sinks {
		sinks[i] = &Source{info: *reply[i]}
	}
	return sinks, nil
}

// DefaultSource returns the default input device.
func (c *Client) DefaultSource() (*Source, error) {
	var source Source
	err := c.c.Request(&proto.GetSourceInfo{SourceIndex: proto.Undefined}, &source.info)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

// Name returns the source name. Source names are always unique, but not necessarily human-readable.
func (s *Source) Name() string {
	return s.info.SourceName
}

// DeviceName is a human-readable name describing the source.
func (s *Source) DeviceName() string {
	return s.info.Device
}
