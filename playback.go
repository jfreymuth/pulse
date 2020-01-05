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

	sinkIndex uint32
	cmap      proto.ChannelMap
	rate      uint32
	bufSize   uint32
	format    byte
}

func (c *Client) NewPlayback(cb interface{}, opts ...PlaybackOption) (*PlaybackStream, error) {
	p := &PlaybackStream{
		c:         c,
		sinkIndex: proto.Undefined,
		cmap:      proto.ChannelMap{proto.ChannelMono},
		rate:      44100,
		bufSize:   proto.Undefined,
	}
	for _, opt := range opts {
		opt(p)
	}

	var format byte
	switch cb := cb.(type) {
	case func([]byte):
		p.format = 0
		p.cb8 = cb
		format = proto.FormatUint8
	case func([]int16):
		p.format = 1
		p.cb16 = cb
		format = formatI16
	case func([]int32):
		p.format = 2
		p.cb32 = cb
		format = formatI32
	case func([]float32):
		p.format = 3
		p.cbf = cb
		format = formatF32
	default:
		panic("pulse: invalid callback type")
	}

	cvol := make(proto.ChannelVolumes, len(p.cmap))
	for i := range cvol {
		cvol[i] = 0x100
	}

	var reply proto.CreatePlaybackStreamReply
	err := c.c.Request(&proto.CreatePlaybackStream{
		SampleSpec:            proto.SampleSpec{Format: format, Channels: byte(len(p.cmap)), Rate: p.rate},
		ChannelMap:            p.cmap,
		SinkIndex:             p.sinkIndex,
		BufferMaxLength:       proto.Undefined,
		Corked:                true,
		BufferTargetLength:    p.bufSize,
		BufferPrebufferLength: proto.Undefined,
		BufferMinimumRequest:  proto.Undefined,
		ChannelVolumes:        cvol,
	}, &reply)
	if err != nil {
		return nil, err
	}
	p.index = reply.StreamIndex
	p.bufSize = reply.BufferTargetLength
	c.mu.Lock()
	c.playback[p.index] = p
	c.mu.Unlock()
	return p, nil
}

func (p *PlaybackStream) buffer(n int) []byte {
	if n > len(p.buf) {
		p.buf = make([]byte, n)
	}
	switch p.format {
	case 0:
		p.cb8(p.buf[:n])
	case 1:
		p.cb16(int16Slice(p.buf[:n]))
	case 2:
		p.cb32(int32Slice(p.buf[:n]))
	case 3:
		p.cbf(float32Slice(p.buf[:n]))
	}
	return p.buf[:n]
}

func (p *PlaybackStream) Start() {
	p.c.c.Request(&proto.FlushPlaybackStream{p.index}, nil)
	p.c.c.Request(&proto.CorkPlaybackStream{p.index, false}, nil)
	p.running = true
	p.c.c.Send(p.index, p.buffer(int(p.bufSize)))
}

func (p *PlaybackStream) Stop() {
	p.c.c.Request(&proto.CorkPlaybackStream{p.index, true}, nil)
	p.running = false
}

func (p *PlaybackStream) Resume() {
	p.c.c.Request(&proto.CorkPlaybackStream{p.index, false}, nil)
	p.running = true
}

func (p *PlaybackStream) Running() bool {
	return p.running
}

func (p *PlaybackStream) SampleRate() int {
	return int(p.rate)
}

func (p *PlaybackStream) Channels() int {
	return len(p.cmap)
}

func (p *PlaybackStream) BufferSize() int {
	s := int(p.bufSize) / len(p.cmap)
	switch p.format {
	case 1:
		s /= 2
	case 2, 3:
		s /= 4
	}
	return s
}

func (p *PlaybackStream) BufferSizeBytes() int {
	return int(p.bufSize)
}

type PlaybackOption func(*PlaybackStream)

var PlaybackMono PlaybackOption = func(p *PlaybackStream) {
	p.cmap = proto.ChannelMap{proto.ChannelMono}
}

var PlaybackStereo PlaybackOption = func(p *PlaybackStream) {
	p.cmap = proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight}
}

func PlaybackChannels(m proto.ChannelMap) PlaybackOption {
	if len(m) == 0 {
		panic("pulse: invalid channel map")
	}
	return func(p *PlaybackStream) {
		p.cmap = m
	}
}

func PlaybackSampleRate(rate int) PlaybackOption {
	return func(p *PlaybackStream) {
		p.rate = uint32(rate)
	}
}

func PlaybackBufferSize(bytes int) PlaybackOption {
	return func(p *PlaybackStream) {
		p.bufSize = uint32(bytes)
	}
}

func PlaybackSinkIndex(index uint32) PlaybackOption {
	return func(p *PlaybackStream) {
		p.sinkIndex = index
	}
}
