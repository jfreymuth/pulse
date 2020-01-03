package pulse

import "github.com/jfreymuth/pulse/proto"

func (c *Client) ListSinks() ([]*proto.GetSinkInfoReply, error) {
	var reply proto.GetSinkInfoListReply
	if err := c.c.Request(&proto.GetSinkInfoList{}, &reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (c *Client) ListSources() ([]*proto.GetSourceInfoReply, error) {
	var reply proto.GetSourceInfoListReply
	if err := c.c.Request(&proto.GetSourceInfoList{}, &reply); err != nil {
		return nil, err
	}
	return reply, nil
}
