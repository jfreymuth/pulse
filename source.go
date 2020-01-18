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

// SourceByID looks up a source id.
func (c *Client) SourceByID(name string) (*Source, error) {
	var source Source
	err := c.c.Request(&proto.GetSourceInfo{SourceIndex: proto.Undefined, SourceName: name}, &source.info)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

// ID returns the source name. Source names are unique identifiers, but not necessarily human-readable.
func (s *Source) ID() string {
	return s.info.SourceName
}

// Name is a human-readable name describing the source.
func (s *Source) Name() string {
	return s.info.Device
}

// Channels returns the default channel map.
func (s *Source) Channels() proto.ChannelMap {
	return s.info.ChannelMap
}

// SampleRate returns the default sample rate.
func (s *Source) SampleRate() int {
	return int(s.info.Rate)
}

// SourceIndex returns the source index.
// This should only be used together with (*Cient).RawRequest.
func (s *Source) SourceIndex() uint32 {
	return s.info.SourceIndex
}
