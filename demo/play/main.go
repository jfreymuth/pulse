package main

import (
	"fmt"
	"math"

	"github.com/jfreymuth/pulse"
)

func main() {
	c, err := pulse.NewClient()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()

	stream, err := c.NewPlayback(synth, pulse.PlaybackLatency(.1))
	if err != nil {
		fmt.Println(err)
		return
	}

	stream.Start()
	stream.Drain()
	fmt.Println("Underflow:", stream.Underflow())
	stream.Close()
}

var t, phase float32

func synth(out []float32) error {
	if t > 4 {
		return pulse.EndOfData
	}
	for i := range out {
		x := float32(math.Sin(2 * math.Pi * float64(phase)))
		out[i] = x * 0.1
		f := [...]float32{440, 550, 660, 880}[int(2*t)&3]
		phase += f / 44100
		if phase >= 1 {
			phase--
		}
		t += 1. / 44100
	}
	return nil
}
