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

	sourceIndex   uint32
	cmap          proto.ChannelMap
	rate          uint32
	format        byte
	fragSize      uint32
	adjustLatency bool
}

func (c *Client) NewRecord(cb interface{}, opts ...RecordOption) (*RecordStream, error) {
	r := &RecordStream{
		c:           c,
		sourceIndex: proto.Undefined,
		cmap:        proto.ChannelMap{proto.ChannelMono},
		rate:        44100,
		fragSize:    proto.Undefined,
	}
	for _, opt := range opts {
		opt(r)
	}

	var format byte
	switch cb := cb.(type) {
	case func([]byte):
		r.format = 0
		r.cb8 = cb
		format = proto.FormatUint8
	case func([]int16):
		r.format = 1
		r.cb16 = cb
		format = formatI16
	case func([]int32):
		r.format = 2
		r.cb32 = cb
		format = formatI32
	case func([]float32):
		r.format = 3
		r.cbf = cb
		format = formatF32
	default:
		panic("pulse: invalid callback type")
	}

	cvol := make(proto.ChannelVolumes, len(r.cmap))
	for i := range cvol {
		cvol[i] = 0x100
	}

	var reply proto.CreateRecordStreamReply
	err := c.c.Request(&proto.CreateRecordStream{
		SampleSpec:         proto.SampleSpec{Format: format, Channels: byte(len(r.cmap)), Rate: r.rate},
		ChannelMap:         r.cmap,
		SourceIndex:        r.sourceIndex,
		BufferMaxLength:    proto.Undefined,
		Corked:             true,
		BufferFragSize:     r.fragSize,
		ChannelVolumes:     cvol,
		DirectOnInputIndex: proto.Undefined,
		AdjustLatency:      r.adjustLatency,
	}, &reply)
	if err != nil {
		return nil, err
	}
	r.index = reply.StreamIndex
	c.mu.Lock()
	c.record[r.index] = r
	c.mu.Unlock()
	return r, nil
}

func (r *RecordStream) write(buf []byte) {
	switch r.format {
	case 0:
		r.cb8(buf)
	case 1:
		r.cb16(int16Slice(buf))
	case 2:
		r.cb32(int32Slice(buf))
	case 3:
		r.cbf(float32Slice(buf))
	}
}

func (r *RecordStream) Start() {
	r.c.c.Request(&proto.FlushRecordStream{r.index}, nil)
	r.c.c.Request(&proto.CorkRecordStream{r.index, false}, nil)
	r.running = true
}

func (r *RecordStream) Stop() {
	r.c.c.Request(&proto.CorkRecordStream{r.index, true}, nil)
	r.running = false
}

func (r *RecordStream) Resume() {
	r.c.c.Request(&proto.CorkRecordStream{r.index, false}, nil)
	r.running = true
}

func (r *RecordStream) Running() bool {
	return r.running
}

func (r *RecordStream) SampleRate() int {
	return int(r.rate)
}

func (r *RecordStream) Channels() int {
	return len(r.cmap)
}

type RecordOption func(*RecordStream)

var RecordMono RecordOption = func(r *RecordStream) {
	r.cmap = proto.ChannelMap{proto.ChannelMono}
}

var RecordStereo RecordOption = func(r *RecordStream) {
	r.cmap = proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight}
}

func RecordChannels(m proto.ChannelMap) RecordOption {
	if len(m) == 0 {
		panic("pulse: invalid channel map")
	}
	return func(r *RecordStream) {
		r.cmap = m
	}
}

func RecordSampleRate(rate int) RecordOption {
	return func(r *RecordStream) {
		r.rate = uint32(rate)
	}
}

func RecordBufferFragmentSize(size uint32) RecordOption {
	return func(r *RecordStream) {
		r.fragSize = size
	}
}

func RecordAdjustLatency(adjust bool) RecordOption {
	return func(r *RecordStream) {
		r.adjustLatency = adjust
	}
}

func RecordSourceIndex(index uint32) RecordOption {
	return func(p *RecordStream) {
		p.sourceIndex = index
	}
}
