package pulse

import (
	"sync"
	"sync/atomic"

	"github.com/jfreymuth/pulse/proto"
)

// A PlaybackStream is used for playing audio.
// When creating a stream, the user must provide a callback that will be used to buffer audio data.
type PlaybackStream struct {
	c *Client

	index     uint32
	state     atomic.Int32
	underflow bool
	err       error

	front, back []byte
	requested   int
	request     chan int
	started     chan bool

	events        chan struct{}
	eventsLock    sync.Mutex
	volumeChanges chan proto.ChannelVolumes

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

	// Listen for changes in the sink input if the application wants to be
	// notified of volume changes.
	if p.volumeChanges != nil {
		p.events = make(chan struct{}, 1)
		go p.handleEvents(p.events)
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
		if streamState(p.state.Load()) != running {
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
				p.state.Store(int32(idle))
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

// Handle events for this playback stream in a goroutine.
// Event notifications are received through the events channel.
func (p *PlaybackStream) handleEvents(events chan struct{}) {
	volume := make(proto.ChannelVolumes, len(p.createRequest.ChannelMap))

	for range events {
		// We got an event that something about our sink input changed, so read
		// the sink input information.
		reply := proto.GetSinkInputInfoReply{}
		err := p.c.c.Request(&proto.GetSinkInputInfo{
			SinkInputIndex: p.index,
		}, &reply)
		if err != nil {
			if p.Closed() {
				// Most likely this error is caused by the stream getting
				// closed. So exit the goroutine.
				break
			}
			// This should not normally happen.
			panic(err)
		}

		// Check whether the volume changed, and if so, report it to the
		// application.
		volumeChanged := false
		for i, val := range reply.ChannelVolumes {
			if volume[i] != val {
				volume[i] = val
				volumeChanged = true
			}
		}
		if volumeChanged {
			volumeToSend := append(proto.ChannelVolumes(nil), volume...) // copy volume

			// Drop last volume change, if not received by the application.
			// This way, if p.volumeChanges is a buffered channel, some updates
			// might get lost when the receiver is slow but it will always
			// receive the latest volume eventually.
			select {
			case <-p.volumeChanges:
				// Dropped, so there was something in the buffered channel.
			default:
				// Not dropped, so if the channel is buffered, it should have
				// room now.
			}

			// Send the new volume.
			select {
			case p.volumeChanges <- volumeToSend:
				// Succeeded in sending!
			default:
				// Somehow couldn't send the new volume value. Perhaps the
				// channel is unbuffered, and the receiving goroutine is doing
				// other things? There's not much we can do about it here.
			}
		}
	}

	// Playback stream was closed, so close the volume changes channel.
	close(p.volumeChanges)
}

// Start starts playing audio.
func (p *PlaybackStream) Start() {
	if p.state.CompareAndSwap(int32(idle), int32(running)) {
		p.c.c.Request(&proto.FlushPlaybackStream{StreamIndex: p.index}, nil)
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
	if !p.state.CompareAndSwap(int32(running), int32(idle)) {
		p.state.CompareAndSwap(int32(paused), int32(idle))
	}
}

// Pause stops playing audio immediately.
func (p *PlaybackStream) Pause() {
	if p.state.CompareAndSwap(int32(running), int32(paused)) {
		p.c.c.Request(&proto.CorkPlaybackStream{StreamIndex: p.index, Corked: true}, nil)
	}
}

// Resume resumes a paused stream.
func (p *PlaybackStream) Resume() {
	if p.state.CompareAndSwap(int32(paused), int32(running)) {
		p.c.c.Request(&proto.CorkPlaybackStream{StreamIndex: p.index, Corked: false}, nil)
		p.underflow = false
	}
}

// Drain waits until the playback has ended.
// Drain does not return when the stream is paused.
func (p *PlaybackStream) Drain() {
	if streamState(p.state.Load()) == running {
		p.c.c.Request(&proto.DrainPlaybackStream{StreamIndex: p.index}, nil)
	}
}

// Volume returns the volume of each channel in the playback.
func (p *PlaybackStream) Volume() (proto.ChannelVolumes, error) {
	reply := proto.GetSinkInputInfoReply{}
	err := p.c.c.Request(&proto.GetSinkInputInfo{
		SinkInputIndex: p.index,
	}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.ChannelVolumes, nil
}

// SetVolume changes the volume of each channel in the playback.
//
// Do not set the volume when opening a playback stream, PulseAudio will pick an
// appropriate volume for the stream automatically (and may save it for next
// time). In particular, don't set it to 100% because depending on the
// configuration this could actually set the volume to the maximum volume the
// hardware is capable of which is usually way too loud. Instead, only change
// the volume as a direct result of user input.
//
// If you use this API, you should also query the volume on startup and listen
// for playback volume changes from the system mixer (many desktops allow users
// to change application volume from the system tray). That way, you can keep
// the volume slider in the application synchronized with the system volume
// mixer.
func (p *PlaybackStream) SetVolume(volumes proto.ChannelVolumes) error {
	return p.c.c.Request(&proto.SetSinkInputVolume{
		SinkInputIndex: p.index,
		ChannelVolumes: volumes,
	}, nil)
}

// Close closes the stream.
func (p *PlaybackStream) Close() {
	if p.state.CompareAndSwap(int32(running), int32(closed)) || p.state.CompareAndSwap(int32(paused), int32(closed)) || p.state.CompareAndSwap(int32(idle), int32(closed)) {
		p.c.c.Request(&proto.DeletePlaybackStream{StreamIndex: p.index}, nil)

		close(p.request)

		close(p.started)

		p.c.mu.Lock()
		delete(p.c.playback, p.index)
		p.c.mu.Unlock()

		p.eventsLock.Lock()
		close(p.events)
		p.events = nil
		p.eventsLock.Unlock()
	}
}

// Closed returns wether the stream was closed.
func (p *PlaybackStream) Closed() bool {
	return streamState(p.state.Load()) == closed
}

// Running returns wether the stream is currently playing.
func (p *PlaybackStream) Running() bool {
	return streamState(p.state.Load()) == running
}

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
	return p.index
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

// PlaybackVolumeChanges sets a channel to receive volume changes on.
// These changes can come either from changing the volume directly (through
// SetVolume) or from the system volume mixer.
//
// The channel should be buffered (1 element is sufficient) to avoid losing
// volume changes due to race conditions or a slow receiver. It will be closed
// when the playback is closed.
func PlaybackVolumeChanges(changes chan proto.ChannelVolumes) PlaybackOption {
	return func(p *PlaybackStream) {
		p.volumeChanges = changes
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
