package pulse

import "github.com/jfreymuth/pulse/proto"

// A PlaybackStream is used for playing audio.
// When creating a stream, the user must provide a callback that will be used to buffer audio data.
type PlaybackStream struct {
	c *Client

	index     uint32
	state     streamState
	underflow bool
	err       error

	front, back []byte
	requested   int
	request     chan int
	started     chan bool

	r Reader

	createRequest  proto.CreatePlaybackStream
	createReply    proto.CreatePlaybackStreamReply
	bytesPerSample int
}

// EndOfData is a special error value that can be returned by a reader to stop the stream.
const EndOfData endOfData = false

// NewPlayback creates a playback stream.
// The created stream wil not be running, it must be started with Start().
// If the reader returns any error, the stream will be stopped. The special error value EndOfData
// can be used to intentionally stop the stream from within the callback.
// The order of options is important in some cases, see the documentation of the individual PlaybackOptions.
func (c *Client) NewPlayback(r Reader, opts ...PlaybackOption) (*PlaybackStream, error) {
	p := &PlaybackStream{
		c: c,
		createRequest: proto.CreatePlaybackStream{
			SinkIndex:             proto.Undefined,
			ChannelMap:            proto.ChannelMap{proto.ChannelMono},
			SampleSpec:            proto.SampleSpec{Format: r.Format(), Channels: 1, Rate: 44100},
			BufferMaxLength:       proto.Undefined,
			Corked:                true,
			BufferTargetLength:    proto.Undefined,
			BufferPrebufferLength: proto.Undefined,
			BufferMinimumRequest:  proto.Undefined,
			Properties:            proto.PropList{},
		},
		bytesPerSample: bytes(r.Format()),
		r:              r,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.createRequest.ChannelVolumes == nil {
		cvol := make(proto.ChannelVolumes, len(p.createRequest.ChannelMap))
		for i := range cvol {
			cvol[i] = 0x100
		}
		p.createRequest.ChannelVolumes = cvol
	}

	err := c.c.Request(&p.createRequest, &p.createReply)
	if err != nil {
		return nil, err
	}
	p.index = p.createReply.StreamIndex
	p.front = make([]byte, p.createReply.BufferMaxLength)
	p.back = make([]byte, p.createReply.BufferMaxLength)
	p.request = make(chan int)
	p.started = make(chan bool)
	c.mu.Lock()
	c.playback[p.index] = p
	c.mu.Unlock()
	go p.run()
	return p, nil
}

func (p *PlaybackStream) run() {
	for n := range p.request {
		if p.state != running {
			continue
		}
		p.requested += n
		for p.requested > 0 {
			n, err := p.r.Read(p.front[:p.requested])
			if n > 0 {
				p.c.c.Send(p.index, p.front[:n])
				p.requested -= n
				p.front, p.back = p.back, p.front
			}
			if err != nil {
				if err != EndOfData {
					p.err = err
				}
				p.state = idle
				break
			}
			select {
			case n = <-p.request:
				p.requested += n
			default:
			}
		}
	}
}

// Start starts playing audio.
func (p *PlaybackStream) Start() {
	if p.state == idle {
		p.c.c.Request(&proto.FlushPlaybackStream{StreamIndex: p.index}, nil)
		p.state = running
		p.err = nil
		p.request <- int(p.createReply.BufferTargetLength)
		p.underflow = false
		p.c.c.Request(&proto.CorkPlaybackStream{StreamIndex: p.index, Corked: false}, nil)
		<-p.started
	}
}

// Stop stops playing audio; the callback will no longer be called.
// If the buffer size/latency is large, audio may continue to play for some time after the call to Stop.
func (p *PlaybackStream) Stop() {
	if p.state == running || p.state == paused {
		p.state = idle
	}
}

// Pause stops playing audio immediately.
func (p *PlaybackStream) Pause() {
	if p.state == running {
		p.c.c.Request(&proto.CorkPlaybackStream{StreamIndex: p.index, Corked: true}, nil)
		p.state = paused
	}
}

// Resume resumes a paused stream.
func (p *PlaybackStream) Resume() {
	if p.state == paused {
		p.c.c.Request(&proto.CorkPlaybackStream{StreamIndex: p.index, Corked: false}, nil)
		p.state = running
		p.underflow = false
	}
}

// Drain waits until the playback has ended.
// Drain does not return when the stream is paused.
func (p *PlaybackStream) Drain() {
	if p.state == running {
		p.c.c.Request(&proto.DrainPlaybackStream{StreamIndex: p.index}, nil)
	}
}

// Close closes the stream.
func (p *PlaybackStream) Close() {
	if !p.Closed() {
		p.c.c.Request(&proto.DeletePlaybackStream{StreamIndex: p.index}, nil)
		p.state = closed
		close(p.request)
		p.c.mu.Lock()
		delete(p.c.playback, p.index)
		p.c.mu.Unlock()
	}
}

// Closed returns wether the stream was closed.
func (p *PlaybackStream) Closed() bool { return p.state == closed || p.state == serverLost }

// Running returns wether the stream is currently playing.
func (p *PlaybackStream) Running() bool { return p.state == running }

// Underflow returns true if any underflows happend since the last call to Start or Resume.
// Underflows usually happen because the latency/buffer size is too low or because the callback
// takes too long to run.
func (p *PlaybackStream) Underflow() bool { return p.underflow }

// Error returns the last error returned by the stream's reader.
func (p *PlaybackStream) Error() error { return p.err }

// SampleRate returns the stream's sample rate (samples per second).
func (p *PlaybackStream) SampleRate() int {
	return int(p.createReply.Rate)
}

// Channels returns the number of channels.
func (p *PlaybackStream) Channels() int {
	return int(p.createReply.Channels)
}

// BufferSize returns the size of the server-side buffer in samples.
func (p *PlaybackStream) BufferSize() int {
	s := int(p.createReply.BufferTargetLength) / int(p.createReply.Channels)
	return s / p.bytesPerSample
}

// BufferSizeBytes returns the size of the server-side buffer in bytes.
func (p *PlaybackStream) BufferSizeBytes() int {
	return int(p.createReply.BufferTargetLength)
}

// StreamIndex returns the stream index.
// This should only be used together with (*Cient).RawRequest.
func (p *PlaybackStream) StreamIndex() uint32 {
	return p.index
}

func (p *PlaybackStream) StreamInputIndex() uint32 {
	return p.createReply.SinkInputIndex
}

// A PlaybackOption supplies configuration when creating streams.
type PlaybackOption func(*PlaybackStream)

// PlaybackMono sets a stream to a single channel.
var PlaybackMono PlaybackOption = func(p *PlaybackStream) {
	p.createRequest.ChannelMap = proto.ChannelMap{proto.ChannelMono}
	p.createRequest.Channels = 1
}

// PlaybackStereo sets a stream to two channels.
var PlaybackStereo PlaybackOption = func(p *PlaybackStream) {
	p.createRequest.ChannelMap = proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight}
	p.createRequest.Channels = 2
}

// PlaybackChannels sets a stream to use a custom channel map.
func PlaybackChannels(m proto.ChannelMap) PlaybackOption {
	if len(m) == 0 {
		panic("pulse: invalid channel map")
	}
	return func(p *PlaybackStream) {
		p.createRequest.ChannelMap = m
		p.createRequest.Channels = byte(len(m))
	}
}

// PlaybackSampleRate sets the stream's sample rate.
func PlaybackSampleRate(rate int) PlaybackOption {
	return func(p *PlaybackStream) {
		p.createRequest.Rate = uint32(rate)
	}
}

// PlaybackBufferSize sets the size of the server-side buffer.
// Setting the buffer size too small causes underflows, resulting in audible artifacts.
//
// Buffer size and latency should not be set at the same time.
func PlaybackBufferSize(samples int) PlaybackOption {
	return func(p *PlaybackStream) {
		p.createRequest.BufferTargetLength = uint32(samples * p.bytesPerSample)
		p.createRequest.AdjustLatency = false
	}
}

// PlaybackLatency sets the stream's latency in seconds.
// Setting the latency too low causes underflows, resulting in audible artifacts.
// Applications should generally use the highest acceptable latency.
//
// This should be set after sample rate and channel options.
//
// Buffer size and latency should not be set at the same time.
func PlaybackLatency(seconds float64) PlaybackOption {
	return func(p *PlaybackStream) {
		p.createRequest.BufferTargetLength = uint32(seconds*float64(p.createRequest.Rate)) * uint32(p.createRequest.Channels) * uint32(p.bytesPerSample)
		p.createRequest.BufferMaxLength = 2 * p.createRequest.BufferTargetLength
		p.createRequest.AdjustLatency = true
	}
}

// PlaybackSink sets the sink the stream should send audio to.
func PlaybackSink(sink *Sink) PlaybackOption {
	return func(p *PlaybackStream) {
		p.createRequest.SinkIndex = sink.info.SinkIndex
	}
}

// PlaybackMediaName sets the streams media name.
// This will e.g. be displayed by a volume control application to identity the stream.
func PlaybackMediaName(name string) PlaybackOption {
	return func(p *PlaybackStream) {
		p.createRequest.Properties["media.name"] = proto.PropListString(name)
	}
}

// PlaybackMediaIconName sets the streams media icon using an xdg icon name.
// This will e.g. be displayed by a volume control application to identity the stream.
func PlaybackMediaIconName(name string) PlaybackOption {
	return func(p *PlaybackStream) {
		p.createRequest.Properties["media.icon_name"] = proto.PropListString(name)
	}
}

// PlaybackRawOption can be used to create custom options.
//
// This is an advanced function, similar to (*Client).RawRequest.
func PlaybackRawOption(o func(*proto.CreatePlaybackStream)) PlaybackOption {
	return func(p *PlaybackStream) {
		o(&p.createRequest)
	}
}

type endOfData bool

func (endOfData) Error() string { return "end of data" }
