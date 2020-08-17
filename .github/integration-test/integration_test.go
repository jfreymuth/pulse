// +build integration

package pulse

import (
	"testing"
	"time"

	"github.com/jfreymuth/pulse"
)

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
