package pulse

import "github.com/jfreymuth/pulse/proto"

type Source struct {
	info proto.GetSourceInfoReply
}

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

func (s *Source) Name() string {
	return s.info.SourceName
}

func (s *Source) DeviceName() string {
	return s.info.Device
}
