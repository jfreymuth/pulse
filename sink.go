package pulse

import "github.com/jfreymuth/pulse/proto"

// A Sink is an output device.
type Sink struct {
	info proto.GetSinkInfoReply
}

// ListSinks returns a list of all available output devices.
func (c *Client) ListSinks() ([]*Sink, error) {
	var reply proto.GetSinkInfoListReply
	err := c.c.Request(&proto.GetSinkInfoList{}, &reply)
	if err != nil {
		return nil, err
	}
	sinks := make([]*Sink, len(reply))
	for i := range sinks {
		sinks[i] = &Sink{info: *reply[i]}
	}
	return sinks, nil
}

// DefaultSink returns the default output device.
func (c *Client) DefaultSink() (*Sink, error) {
	var sink Sink
	err := c.c.Request(&proto.GetSinkInfo{SinkIndex: proto.Undefined}, &sink.info)
	if err != nil {
		return nil, err
	}
	return &sink, nil
}

// SinkByID looks up a sink id.
func (c *Client) SinkByID(name string) (*Sink, error) {
	var sink Sink
	err := c.c.Request(&proto.GetSinkInfo{SinkIndex: proto.Undefined, SinkName: name}, &sink.info)
	if err != nil {
		return nil, err
	}
	return &sink, nil
}

// ID returns the sink name. Sink names are unique identifiers, but not necessarily human-readable.
func (s *Sink) ID() string {
	return s.info.SinkName
}

// Name is a human-readable name describing the sink.
func (s *Sink) Name() string {
	return s.info.Device
}

// Channels returns the default channel map.
func (s *Sink) Channels() proto.ChannelMap {
	return s.info.ChannelMap
}

// SampleRate returns the default sample rate.
func (s *Sink) SampleRate() int {
	return int(s.info.Rate)
}

// SinkIndex returns the sink index.
// This should only be used together with (*Cient).RawRequest.
func (s *Sink) SinkIndex() uint32 {
	return s.info.SinkIndex
}
