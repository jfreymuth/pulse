package main

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

func run(inpath string) error {
	f, err := os.Open(inpath)
	if err != nil {
		return err
	}
	defer f.Close()

	d, err := mp3.NewDecoder(f)
	if err != nil {
		return err
	}

	c, err := pulse.NewClient()
	if err != nil {
		return err
	}
	defer c.Close()

	cb := func(out []int16) {
		var sample [2]byte

		for i := range out {
			if _, err := d.Read(sample[:]); err != nil {
				fmt.Println(err)
				return
			}

			v := int(sample[0]) | (int(sample[1]) << 8)

			out[i] = int16(v)

		}
	}

	stream, err := c.NewPlayback(cb,
		pulse.PlaybackSampleRate(d.SampleRate()),
		pulse.PlaybackChannels(proto.ChannelMap{
			proto.ChannelLeft,
			proto.ChannelRight,
		}))
	if err != nil {
		return err
	}

	fmt.Printf("Playing %#v ...\n", inpath)
	stream.Start()

	// TODO: How to tell the library we've hit the end?
	select {}

	return nil
}

func main() {
	for _, inpath := range os.Args[1:] {
		if err := run(inpath); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
