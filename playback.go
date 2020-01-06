package pulse

import "github.com/jfreymuth/pulse/proto"

type PlaybackStream struct {
	c       *Client
	index   uint32
	running bool

	buf []byte

	cb8  func([]byte)
	cb16 func([]int16)
	cb32 func([]int32)
	cbf  func([]float32)

	createRequest proto.CreatePlaybackStream
	createReply   proto.CreatePlaybackStreamReply
}

func (c *Client) NewPlayback(cb interface{}, opts ...PlaybackOption) (*PlaybackStream, error) {
	p := &PlaybackStream{
		c: c,
		createRequest: proto.CreatePlaybackStream{
			SinkIndex:             proto.Undefined,
			ChannelMap:            proto.ChannelMap{proto.ChannelMono},
			SampleSpec:            proto.SampleSpec{Channels: 1, Rate: 44100},
			BufferMaxLength:       proto.Undefined,
			Corked:                true,
			BufferTargetLength:    proto.Undefined,
			BufferPrebufferLength: proto.Undefined,
			BufferMinimumRequest:  proto.Undefined,
		},
	}
	for _, opt := range opts {
		opt(p)
	}

	switch cb := cb.(type) {
	case func([]byte):
		p.cb8 = cb
		p.createRequest.Format = proto.FormatUint8
	case func([]int16):
		p.cb16 = cb
		p.createRequest.Format = formatI16
	case func([]int32):
		p.cb32 = cb
		p.createRequest.Format = formatI32
	case func([]float32):
		p.cbf = cb
		p.createRequest.Format = formatF32
	default:
		panic("pulse: invalid callback type")
	}

	cvol := make(proto.ChannelVolumes, len(p.createRequest.ChannelMap))
	for i := range cvol {
		cvol[i] = 0x100
	}
	p.createRequest.ChannelVolumes = cvol

	err := c.c.Request(&p.createRequest, &p.createReply)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.playback[p.index] = p
	c.mu.Unlock()
	return p, nil
}

func (p *PlaybackStream) buffer(n int) []byte {
	if n > len(p.buf) {
		p.buf = make([]byte, n)
	}
	switch {
	case p.cb8 != nil:
		p.cb8(p.buf[:n])
	case p.cb16 != nil:
		p.cb16(int16Slice(p.buf[:n]))
	case p.cb32 != nil:
		p.cb32(int32Slice(p.buf[:n]))
	case p.cbf != nil:
		p.cbf(float32Slice(p.buf[:n]))
	}
	return p.buf[:n]
}

func (p *PlaybackStream) Start() {
	p.c.c.Request(&proto.FlushPlaybackStream{StreamIndex: p.index}, nil)
	p.c.c.Request(&proto.CorkPlaybackStream{StreamIndex: p.index, Corked: false}, nil)
	p.running = true
	p.c.c.Send(p.index, p.buffer(int(p.createReply.BufferTargetLength)))
}

func (p *PlaybackStream) Stop() {
	p.c.c.Request(&proto.CorkPlaybackStream{StreamIndex: p.index, Corked: true}, nil)
	p.running = false
}

func (p *PlaybackStream) Resume() {
	p.c.c.Request(&proto.CorkPlaybackStream{StreamIndex: p.index, Corked: false}, nil)
	p.running = true
}

func (p *PlaybackStream) Running() bool {
	return p.running
}

func (p *PlaybackStream) SampleRate() int {
	return int(p.createReply.Rate)
}

func (p *PlaybackStream) Channels() int {
	return int(p.createReply.Channels)
}

func (p *PlaybackStream) BufferSize() int {
	s := int(p.createReply.BufferTargetLength) / int(p.createReply.Channels)
	switch {
	case p.cb16 != nil:
		s /= 2
	case p.cb8 == nil:
		s /= 4
	}
	return s
}

func (p *PlaybackStream) BufferSizeBytes() int {
	return int(p.createReply.BufferTargetLength)
}

type PlaybackOption func(*PlaybackStream)

var PlaybackMono PlaybackOption = func(p *PlaybackStream) {
	p.createRequest.ChannelMap = proto.ChannelMap{proto.ChannelMono}
	p.createRequest.Channels = 1
}

var PlaybackStereo PlaybackOption = func(p *PlaybackStream) {
	p.createRequest.ChannelMap = proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight}
	p.createRequest.Channels = 2
}

func PlaybackChannels(m proto.ChannelMap) PlaybackOption {
	if len(m) == 0 {
		panic("pulse: invalid channel map")
	}
	return func(p *PlaybackStream) {
		p.createRequest.ChannelMap = m
		p.createRequest.Channels = byte(len(m))
	}
}

func PlaybackSampleRate(rate int) PlaybackOption {
	return func(p *PlaybackStream) {
		p.createRequest.Rate = uint32(rate)
	}
}

func PlaybackBufferSize(bytes int) PlaybackOption {
	return func(p *PlaybackStream) {
		p.createRequest.BufferTargetLength = uint32(bytes)
	}
}

func PlaybackSinkIndex(index uint32) PlaybackOption {
	return func(p *PlaybackStream) {
		p.createRequest.SinkIndex = index
	}
}
