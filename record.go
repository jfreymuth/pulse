package pulse

import "github.com/jfreymuth/pulse/proto"

type RecordStream struct {
	c       *Client
	index   uint32
	running bool

	cb8  func([]byte)
	cb16 func([]int16)
	cb32 func([]int32)
	cbf  func([]float32)

	createRequest proto.CreateRecordStream
	createReply   proto.CreateRecordStreamReply
}

func (c *Client) NewRecord(cb interface{}, opts ...RecordOption) (*RecordStream, error) {
	r := &RecordStream{
		c: c,
		createRequest: proto.CreateRecordStream{
			SourceIndex:        proto.Undefined,
			ChannelMap:         proto.ChannelMap{proto.ChannelMono},
			SampleSpec:         proto.SampleSpec{Channels: 1, Rate: 44100},
			BufferMaxLength:    proto.Undefined,
			Corked:             true,
			BufferFragSize:     proto.Undefined,
			DirectOnInputIndex: proto.Undefined,
		},
	}
	for _, opt := range opts {
		opt(r)
	}

	switch cb := cb.(type) {
	case func([]byte):
		r.cb8 = cb
		r.createRequest.Format = proto.FormatUint8
	case func([]int16):
		r.cb16 = cb
		r.createRequest.Format = formatI16
	case func([]int32):
		r.cb32 = cb
		r.createRequest.Format = formatI32
	case func([]float32):
		r.cbf = cb
		r.createRequest.Format = formatF32
	default:
		panic("pulse: invalid callback type")
	}

	cvol := make(proto.ChannelVolumes, len(r.createRequest.ChannelMap))
	for i := range cvol {
		cvol[i] = 0x100
	}

	err := c.c.Request(&r.createRequest, &r.createReply)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.record[r.index] = r
	c.mu.Unlock()
	return r, nil
}

func (r *RecordStream) write(buf []byte) {
	switch {
	case r.cb8 != nil:
		r.cb8(buf)
	case r.cb16 != nil:
		r.cb16(int16Slice(buf))
	case r.cb32 != nil:
		r.cb32(int32Slice(buf))
	case r.cbf != nil:
		r.cbf(float32Slice(buf))
	}
}

func (r *RecordStream) Start() {
	r.c.c.Request(&proto.FlushRecordStream{StreamIndex: r.index}, nil)
	r.c.c.Request(&proto.CorkRecordStream{StreamIndex: r.index, Corked: false}, nil)
	r.running = true
}

func (r *RecordStream) Stop() {
	r.c.c.Request(&proto.CorkRecordStream{StreamIndex: r.index, Corked: true}, nil)
	r.running = false
}

func (r *RecordStream) Resume() {
	r.c.c.Request(&proto.CorkRecordStream{StreamIndex: r.index, Corked: false}, nil)
	r.running = true
}

func (r *RecordStream) Running() bool {
	return r.running
}

func (r *RecordStream) SampleRate() int {
	return int(r.createReply.Rate)
}

func (r *RecordStream) Channels() int {
	return int(r.createReply.Channels)
}

type RecordOption func(*RecordStream)

var RecordMono RecordOption = func(r *RecordStream) {
	r.createRequest.ChannelMap = proto.ChannelMap{proto.ChannelMono}
	r.createRequest.Channels = 1
}

var RecordStereo RecordOption = func(r *RecordStream) {
	r.createRequest.ChannelMap = proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight}
	r.createRequest.Channels = 2
}

func RecordChannels(m proto.ChannelMap) RecordOption {
	if len(m) == 0 {
		panic("pulse: invalid channel map")
	}
	return func(r *RecordStream) {
		r.createRequest.ChannelMap = m
		r.createRequest.Channels = byte(len(m))
	}
}

func RecordSampleRate(rate int) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.Rate = uint32(rate)
	}
}

func RecordBufferFragmentSize(size uint32) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.BufferFragSize = size
	}
}

func RecordAdjustLatency(adjust bool) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.AdjustLatency = adjust
	}
}

func RecordSourceIndex(index uint32) RecordOption {
	return func(p *RecordStream) {
		p.createRequest.SourceIndex = index
	}
}
