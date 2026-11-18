//go:build integration
// +build integration

package pulse

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jfreymuth/pulse"
)

func TestPlaybackStartWithEmptySource(t *testing.T) {
	client, err := pulse.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	stream, err := client.NewPlayback(pulse.Int16Reader(func([]int16) (int, error) {
		return 0, pulse.EndOfData
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	// Each start must observe the result for its own prebuffer attempt.
	for i := 0; i < 2; i++ {
		startPlayback(t, stream)
		if stream.Running() || stream.Error() != nil {
			t.Fatalf("after EndOfData: running=%t, error=%v", stream.Running(), stream.Error())
		}
	}
}

func beginPlayback(stream *pulse.PlaybackStream) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		stream.Start()
		close(done)
	}()
	return done
}

// Bound startup independently of the go test timeout so a broken start
// identifies the failing transition.
func startPlayback(t *testing.T, stream *pulse.PlaybackStream) {
	t.Helper()
	done := beginPlayback(stream)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("playback Start did not return")
	}
}

func TestPlaybackStartAfterPartialBuffer(t *testing.T) {
	failure := errors.New("test reader failed")
	for _, tc := range []struct {
		name             string
		readErr, wantErr error
	}{
		{"end_of_data", pulse.EndOfData, nil},
		{"reader_error", failure, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testPlaybackRetry(t, tc.readErr, tc.wantErr)
		})
	}
}

func testPlaybackRetry(t *testing.T, readErr, wantErr error) {
	t.Helper()
	client, err := pulse.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Listen for the square wave from the second start. The partial buffer
	// from the first start contains silence.
	heard := make(chan struct{}, 1)
	record, err := client.NewRecord(pulse.Int16Writer(func(in []int16) (int, error) {
		for _, sample := range in {
			if sample == 1000 || sample == -1000 {
				select {
				case heard <- struct{}{}:
				default:
				}
				break
			}
		}
		return len(in), nil
	}), pulse.RecordBufferFragmentSize(256))
	if err != nil {
		t.Fatal(err)
	}
	defer record.Close()
	record.Start()

	reads := 0
	gen := &sampleWaveGenerator{}
	stream, err := client.NewPlayback(pulse.Int16Reader(func(out []int16) (int, error) {
		reads++
		switch reads {
		case 1:
			out[0] = 0
			return 1, nil
		case 2:
			return 0, readErr
		default:
			return gen.generate(out)
		}
	}), pulse.PlaybackBufferSize(256))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	startPlayback(t, stream)
	if stream.Running() || stream.Error() != wantErr {
		t.Fatalf("after partial buffer: running=%t, error=%v", stream.Running(), stream.Error())
	}
	startPlayback(t, stream)
	if !stream.Running() || stream.Error() != nil {
		t.Fatalf("after restart: running=%t, error=%v", stream.Running(), stream.Error())
	}
	select {
	case <-heard:
	case <-time.After(5 * time.Second):
		t.Fatal("no generated audio recorded after restarting the same stream")
	}

	// Resume can generate another Started notification on the same stream.
	// A subsequent request must still get a reply from the protocol loop.
	stream.Pause()
	stream.Resume()
	if !stream.Running() {
		t.Fatal("stream did not resume after retry")
	}
	if _, err := stream.Volume(); err != nil {
		t.Fatalf("after resume: %v", err)
	}
}

func TestStopDuringPlaybackStart(t *testing.T) {
	client, err := pulse.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	entered := make(chan struct{})
	readCtx, release := context.WithCancel(context.Background())
	firstRead := true
	stream, err := client.NewPlayback(pulse.Int16Reader(func(out []int16) (int, error) {
		if firstRead {
			firstRead = false
			close(entered)
			<-readCtx.Done()
			return 0, pulse.EndOfData
		}
		return len(out), nil
	}), pulse.PlaybackBufferSize(256))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	defer release()

	done := beginPlayback(stream)
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("reader did not start prebuffering")
	}
	stream.Stop()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop left Start waiting for the server")
	}
	if stream.Running() {
		t.Fatal("stopped stream is still running")
	}

	restarted := beginPlayback(stream)
	deadline := time.Now().Add(5 * time.Second)
	for !stream.Running() {
		if time.Now().After(deadline) {
			t.Fatal("second startup did not enter prebuffering")
		}
		time.Sleep(time.Millisecond)
	}
	// Complete the old read only after the new startup has claimed the stream.
	release()
	select {
	case <-restarted:
	case <-time.After(5 * time.Second):
		t.Fatal("second startup did not complete")
	}
	if !stream.Running() {
		t.Fatal("old reader completion stopped the new playback")
	}
}

func TestLoopback(t *testing.T) {
	var buf []int16

	// Dummy loopback doesn't start immediately sometimes.
	// Retry a few times.
	for retry := 0; retry < 3; retry++ {
		buf = nil
		c, err := pulse.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		record, err := c.NewRecord(
			pulse.Int16Writer(func(in []int16) (int, error) {
				buf = append(buf, in...)
				return len(in), nil
			}),
			pulse.RecordBufferFragmentSize(256),
		)
		if err != nil {
			t.Fatal(err)
		}

		playback, err := c.NewPlayback(
			pulse.Int16Reader((&sampleWaveGenerator{}).generate),
			pulse.PlaybackBufferSize(256),
		)
		if err != nil {
			t.Fatal(err)
		}

		record.Start()
		playback.Start()
		time.Sleep(time.Second)

		record.Stop()
		playback.Stop()
		playback.Drain()

		if len(buf) > 1024 {
			break
		}
	}
	assertSamples(t, buf)
}

func TestUnderflow(t *testing.T) {
	var buf []int16

	// Dummy loopback doesn't start immediately sometimes.
	// Retry a few times.
	for retry := 0; retry < 3; retry++ {
		buf = nil
		c, err := pulse.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		unblocked := make(chan struct{})
		enableRecord := make(chan struct{})

		record, err := c.NewRecord(
			pulse.Int16Writer(func(in []int16) (int, error) {
				select {
				case <-enableRecord:
					buf = append(buf, in...)
				default:
				}
				return len(in), nil
			}),
			pulse.RecordBufferFragmentSize(256),
		)
		if err != nil {
			t.Fatal(err)
		}

		var initPlay bool
		gen := &sampleWaveGenerator{}
		playback, err := c.NewPlayback(
			pulse.Int16Reader(func(out []int16) (int, error) {
				// Return the first buffer immediately to unblock Start()
				if initPlay {
					<-unblocked
				}
				initPlay = true
				return gen.generate(out)
			}),
			pulse.PlaybackBufferSize(256),
		)
		if err != nil {
			t.Fatal(err)
		}

		record.Start()
		playback.Start()

		time.Sleep(500 * time.Millisecond)

		// Unblock reader
		close(unblocked)

		// Skip data during block
		time.Sleep(500 * time.Millisecond)
		close(enableRecord)

		time.Sleep(time.Second)

		if !playback.Underflow() {
			t.Error("Playback should be marked underflow if reader had been blocked")
		}

		record.Stop()
		playback.Stop()
		playback.Drain()

		if len(buf) > 1024 {
			break
		}
	}
	assertSamples(t, buf)
}

type sampleWaveGenerator struct {
	cnt uint32
}

func (g *sampleWaveGenerator) generate(out []int16) (int, error) {
	// Play rectangular wave
	for i, _ := range out {
		if g.cnt%16 < 8 {
			out[i] = 1000
		} else {
			out[i] = -1000
		}
		g.cnt++
	}
	return len(out), nil
}

func assertSamples(t *testing.T, buf []int16) {
	if len(buf) < 1024 {
		t.Fatalf("Could not record enough number of the samples. (%d)", len(buf))
	}

	// Check recorded signal
	var histogram [3]int
	for _, v := range buf {
		switch {
		case v == -1000:
			histogram[0]++
		case v == 1000:
			histogram[2]++
		default:
			histogram[1]++
		}
	}
	if histogram[1] > len(buf)/100 {
		t.Errorf("Recorded signal has values not in played signal.")
	}
	dutyRatio := float32(histogram[2]) / float32(histogram[0]+histogram[2])
	if dutyRatio < 0.49 || 0.51 < dutyRatio {
		t.Errorf("Duty ratio of the recorded signal is not matched to the played signal. (played: 0.5, recorded: %f)", dutyRatio)
	}
}
