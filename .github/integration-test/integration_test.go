// +build integration

package pulse

import (
	"testing"
	"time"

	"github.com/jfreymuth/pulse"
)

func TestIntegration(t *testing.T) {
	var buf []int16

	// Dummy loopback doesn't start immediately sometimes.
	// Retry a few times.
	for retry := 0; retry < 3; retry++ {
		c, err := pulse.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		record, err := c.NewRecord(
			func(in []int16) {
				buf = append(buf, in...)
			},
			pulse.RecordBufferFragmentSize(256),
		)
		if err != nil {
			t.Fatal(err)
		}

		var cnt uint32
		playback, err := c.NewPlayback(
			pulse.Int16Reader(func(out []int16) (int, error) {
				// Play rectangular wave
				for i, _ := range out {
					if cnt%16 < 8 {
						out[i] = 1000
					} else {
						out[i] = -1000
					}
					cnt++
				}
				return len(out), nil
			}),
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

		if len(buf) > 256 {
			break
		}
	}

	if len(buf) < 256 {
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
