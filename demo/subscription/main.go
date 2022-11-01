package main

import (
	"flag"
	"log"

	"github.com/jfreymuth/pulse/proto"
)

func main() {
	sinkName := flag.String("sink", "@DEFAULT_SINK@", "Sink name to watch volume changes")

	flag.Parse()

	client, conn, err := proto.Connect("")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ch := make(chan struct{}, 1)

	client.Callback = func(val interface{}) {
		switch val := val.(type) {
		case *proto.SubscribeEvent:
			log.Printf("%s index=%d", val.Event, val.Index)
			if val.Event.GetType() == proto.EventChange && val.Event.GetFacility() == proto.EventSink {
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}

	props := proto.PropList{}
	err = client.Request(&proto.SetClientName{Props: props}, nil)
	if err != nil {
		panic(err)
	}

	err = client.Request(&proto.Subscribe{Mask: proto.SubscriptionMaskAll}, nil)
	if err != nil {
		panic(err)
	}

	for {
		<-ch
		repl := proto.GetSinkInfoReply{}
		err = client.Request(&proto.GetSinkInfo{SinkIndex: proto.Undefined, SinkName: *sinkName}, &repl)
		if err != nil {
			panic(err)
		}
		var acc int64
		for _, vol := range repl.ChannelVolumes {
			acc += int64(vol)
		}
		acc /= int64(len(repl.ChannelVolumes))
		pct := float64(acc) / float64(proto.VolumeNorm) * 100.0
		log.Printf("%s volume: %.0f%%", *sinkName, pct)
	}
}
