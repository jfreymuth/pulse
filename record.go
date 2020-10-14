package pulse

import "github.com/jfreymuth/pulse/proto"

// A RecordStream is used for recording audio.
// When creating a stream, the user must provide a callback that will be called with the recorded audio data.
type RecordStream struct {
	c *Client

	index uint32
	state streamState
	err   error

	w Writer

	createRequest  proto.CreateRecordStream
	createReply    proto.CreateRecordStreamReply
	bytesPerSample int
}

// NewRecord creates a record stream.
// If the reader returns any error, the stream will be stopped.
// The created stream wil not be running, it must be started with Start().
// The order of options is important in some cases, see the documentation of the individual RecordOptions.
func (c *Client) NewRecord(w Writer, opts ...RecordOption) (*RecordStream, error) {
	r := &RecordStream{
		c: c,
		createRequest: proto.CreateRecordStream{
			SourceIndex:        proto.Undefined,
			ChannelMap:         proto.ChannelMap{proto.ChannelMono},
			SampleSpec:         proto.SampleSpec{Format: w.Format(), Channels: 1, Rate: 44100},
			BufferMaxLength:    proto.Undefined,
			Corked:             true,
			BufferFragSize:     proto.Undefined,
			DirectOnInputIndex: proto.Undefined,
			Properties:         proto.PropList{},
		},
		bytesPerSample: bytes(w.Format()),
		w:              w,
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.createRequest.ChannelVolumes == nil {
		cvol := make(proto.ChannelVolumes, len(r.createRequest.ChannelMap))
		for i := range cvol {
			cvol[i] = 0x100
		}
		r.createRequest.ChannelVolumes = cvol
	}

	err := c.c.Request(&r.createRequest, &r.createReply)
	if err != nil {
		return nil, err
	}
	r.index = r.createReply.StreamIndex
	c.mu.Lock()
	c.record[r.index] = r
	c.mu.Unlock()
	return r, nil
}

func (r *RecordStream) write(buf []byte) {
	if r.err != nil {
		return
	}
	_, err := r.w.Write(buf)
	if err != nil {
		r.err = err
		go r.Stop()
	}
}

// Start starts recording audio.
func (r *RecordStream) Start() {
	if r.state == idle {
		r.err = nil
		r.c.c.Request(&proto.FlushRecordStream{StreamIndex: r.index}, nil)
		r.c.c.Request(&proto.CorkRecordStream{StreamIndex: r.index, Corked: false}, nil)
		r.state = running
	}
}

// Stop stops recording audio; the callback will no longer be called.
func (r *RecordStream) Stop() {
	if r.state == running {
		r.c.c.Request(&proto.CorkRecordStream{StreamIndex: r.index, Corked: true}, nil)
		r.state = idle
	}
}

// Close closes the stream.
func (r *RecordStream) Close() {
	if !r.Closed() {
		r.c.c.Request(&proto.DeleteRecordStream{StreamIndex: r.index}, nil)
		r.state = closed
		r.c.mu.Lock()
		delete(r.c.record, r.index)
		r.c.mu.Unlock()
	}
}

// Closed returns wether the stream was closed.
// Calling other methods on a closed stream may panic.
func (r *RecordStream) Closed() bool { return r.state == closed || r.state == serverLost }

// Running returns wether the stream is currently recording.
func (r *RecordStream) Running() bool { return r.state == running }

// Error returns the last error returned by the stream's writer.
func (r *RecordStream) Error() error { return r.err }

// SampleRate returns the stream's sample rate (samples per second).
func (r *RecordStream) SampleRate() int {
	return int(r.createReply.Rate)
}

// Channels returns the number of channels.
func (r *RecordStream) Channels() int {
	return int(r.createReply.Channels)
}

// StreamIndex returns the stream index.
// This should only be used together with (*Cient).RawRequest.
func (r *RecordStream) StreamIndex() uint32 {
	return r.index
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
		r.createRequest.Properties["media.name"] = proto.PropListString(name)
	}
}

// RecordMediaIconName sets the streams media icon using an xdg icon name.
// This will e.g. be displayed by a volume control application to identity the stream.
func RecordMediaIconName(name string) RecordOption {
	return func(r *RecordStream) {
		r.createRequest.Properties["media.icon_name"] = proto.PropListString(name)
	}
}

// RecordRawOption can be used to create custom options.
//
// This is an advanced function, similar to (*Client).RawRequest.
func RecordRawOption(o func(*proto.CreateRecordStream)) RecordOption {
	return func(p *RecordStream) {
		o(&p.createRequest)
	}
}
