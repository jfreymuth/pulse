package pulse

import "github.com/jfreymuth/pulse/proto"

type Sink struct {
	info proto.GetSinkInfoReply
}

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

func (c *Client) DefaultSink() (*Sink, error) {
	var sink Sink
	err := c.c.Request(&proto.GetSinkInfo{SinkIndex: proto.Undefined}, &sink.info)
	if err != nil {
		return nil, err
	}
	return &sink, nil
}

func (s *Sink) Name() string {
	return s.info.SinkName
}

func (s *Sink) DeviceName() string {
	return s.info.Device
}

func (s *Sink) Channels() proto.ChannelMap {
	return s.info.ChannelMap
}

func (s *Sink) SampleRate() int {
	return int(s.info.Rate)
}
