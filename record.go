package pulse

import "github.com/jfreymuth/pulse/proto"

// A RecordStream is used for recording audio.
// When creating a stream, the user must provide a callback that will be called with the recorded audio data.
type RecordStream struct {
	c       *Client
	index   uint32
	running bool

	bytesPerSample int

	cb8  func([]byte)
	cb16 func([]int16)
	cb32 func([]int32)
	cbf  func([]float32)

	createRequest proto.CreateRecordStream
	createReply   proto.CreateRecordStreamReply
}

// NewRecord creates a record stream.
// The type of cb must be func([]byte), func([]int16), func([]int32), or func([]float32).
// The created stream wil not be running, it must be started with Start().
// The order of options is important in some cases, see the documentation of the individual RecordOptions.
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
			Properties:         map[string]string{},
		},
	}

	switch cb := cb.(type) {
	case func([]byte):
		r.cb8 = cb
		r.createRequest.Format = proto.FormatUint8
		r.bytesPerSample = 1
	case func([]int16):
		r.cb16 = cb
		r.createRequest.Format = formatI16
		r.bytesPerSample = 2
	case func([]int32):
		r.cb32 = cb
		r.createRequest.Format = formatI32
		r.bytesPerSample = 4
	case func([]float32):
		r.cbf = cb
		r.createRequest.Format = formatF32
		r.bytesPerSample = 4
	default:
		panic("pulse: invalid callback type")
	}

	for _, opt := range opts {
		opt(r)
	}

	cvol := make(proto.ChannelVolumes, len(r.createRequest.ChannelMap))
	for i := range cvol {
		cvol[i] = 0x100
	}
	r.createRequest.ChannelVolumes = cvol

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

// Start starts recording audio.
func (r *RecordStream) Start() {
	r.c.c.Request(&proto.FlushRecordStream{StreamIndex: r.index}, nil)
	r.c.c.Request(&proto.CorkRecordStream{StreamIndex: r.index, Corked: false}, nil)
	r.running = true
}

// Stop stops recording audio; the callback will no longer be called.
func (r *RecordStream) Stop() {
	r.c.c.Request(&proto.CorkRecordStream{StreamIndex: r.index, Corked: true}, nil)
	r.running = false
}

// Resume resumes a stopped stream.
func (r *RecordStream) Resume() {
	r.c.c.Request(&proto.CorkRecordStream{StreamIndex: r.index, Corked: false}, nil)
	r.running = true
}

// Close closes the stream.
// Calling methods on a closed stream may panic.
func (r *RecordStream) Close() {
	r.c.c.Request(&proto.DeleteRecordStream{StreamIndex: r.index}, nil)
	r.running = false
	r.c = nil
}

// Closed returns wether the stream was closed.
// Calling other methods on a closed stream may panic.
func (r *RecordStream) Closed() bool {
	return r.c == nil
}

// Running returns wether the stream is currently recording.
func (r *RecordStream) Running() bool {
	return r.running
}

// SampleRate returns the stream's sample rate (samples per second).
func (r *RecordStream) SampleRate() int {
	return int(r.createReply.Rate)
}

// Channels returns the number of channels.
func (r *RecordStream) Channels() int {
	return int(r.createReply.Channels)
}

// A RecordOption supplies configuration when creating streams.
type RecordOption func(*RecordStream)

// RecordMono sets a stream to a single channel.
var RecordMono RecordOption = func(r *RecordStream) {
	r.createRequest.ChannelMap = proto.ChannelMap{proto.ChannelMono}
	r.createRequest.Channels = 1
}

// RecordStereo sets a stream to two channels.
var RecordStereo RecordOption = func(r *RecordStream) {
	r.createRequest.ChannelMap = proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight}
	r.createRequest.Channels = 2
}

// RecordChannels sets a stream to use a custom channel map.
func RecordChannels(m proto.ChannelMap) RecordOption {
	if len(m) == 0 {
		panic("pulse: invalid channel map")
	}
	return func(r *RecordStream) {
		r.createRequest.ChannelMap = m
		r.createRequest.Channels = byte(len(m))
	}
}

// RecordSampleRate sets the stream's sample rate.
func RecordSampleRate(rate int) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.Rate = uint32(rate)
	}
}

// RecordBufferFragmentSize sets the fragment size. This is the size (in bytes) of the buffer passed to the callback.
// Lower values reduce latency, at the cost of more overhead.
//
// Fragment size and latency should not be set at the same time.
func RecordBufferFragmentSize(size uint32) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.BufferFragSize = size
		r.createRequest.AdjustLatency = false
	}
}

// RecordLatency sets the stream's latency in seconds.
//
// This should be set after sample rate and channel options.
//
// Fragment size and latency should not be set at the same time.
func RecordLatency(seconds float64) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.BufferFragSize = uint32(seconds*float64(r.createRequest.Rate)) * uint32(r.createRequest.Channels) * uint32(r.bytesPerSample)
		r.createRequest.BufferMaxLength = 2 * r.createRequest.BufferFragSize
		r.createRequest.AdjustLatency = true
	}
}

// RecordSource sets the source the stream should receive audio from.
func RecordSource(source *Source) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.SourceIndex = source.info.SourceIndex
	}
}

// RecordMonitor sets the stream to receive audio sent to the sink.
func RecordMonitor(sink *Sink) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.SourceIndex = sink.info.MonitorSourceIndex
	}
}

// RecordMediaName sets the streams media name.
// This will e.g. be displayed by a volume control application to identity the stream.
func RecordMediaName(name string) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.Properties["media.name"] = name
	}
}

// RecordMediaIconName sets the streams media icon using an xdg icon name.
// This will e.g. be displayed by a volume control application to identity the stream.
func RecordMediaIconName(name string) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.Properties["media.icon_name"] = name
	}
}
