//go:build integration
// +build integration

package pulse_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

func TestPlaybackErrorOnServerDisconnect(t *testing.T) {
	c, err := pulse.NewClient(pulse.ClientApplicationName("playback-error-integration"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	p, err := c.NewPlayback(pulse.Float32Reader(func(samples []float32) (int, error) { return len(samples), nil }))
	if err != nil {
		t.Fatal(err)
	}
	stopPolling := make(chan struct{})
	pollingDone := make(chan struct{})
	go func() {
		defer close(pollingDone)
		for {
			select {
			case <-stopPolling:
				return
			default:
				_ = p.Error()
				runtime.Gosched()
			}
		}
	}()
	defer func() { close(stopPolling); <-pollingDone }()
	wait := func(stage string, condition func() bool) {
		deadline := time.Now().Add(5 * time.Second)
		for !condition() {
			if time.Now().After(deadline) {
				t.Fatalf("timed out waiting for %s", stage)
			}
			time.Sleep(time.Millisecond)
		}
	}
	control, err := pulse.NewClient(pulse.ClientApplicationName("playback-error-controller"))
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	var clients proto.GetClientInfoListReply
	if err := control.RawRequest(&proto.GetClientInfoList{}, &clients); err != nil {
		t.Fatal(err)
	}
	var index uint32
	found := false
	for _, client := range clients {
		if client.Application == "playback-error-integration" {
			index = client.ClientIndex
			found = true
			break
		}
	}
	if !found {
		t.Fatal("test client missing from server")
	}
	if err := control.RawRequest(&proto.KillClient{ClientIndex: index}, nil); err != nil {
		t.Fatal(err)
	}
	wait("disconnect", p.Closed)
	if got := p.Error(); got != pulse.ErrConnectionClosed {
		t.Fatalf("disconnect error = %v", got)
	}
}
