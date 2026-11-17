package pulse

import (
	"encoding/binary"
	"errors"
	"github.com/jfreymuth/pulse/proto"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestPlaybackErrorConcurrentReader(t *testing.T) {
	want := errors.New("reader failed")
	for _, result := range []error{want, EndOfData} {
		t.Run(result.Error(), func(t *testing.T) {
			start := make(chan struct{})
			p := &PlaybackStream{
				r:       Float32Reader(func([]float32) (int, error) { <-start; return 0, result }),
				state:   newStateMachine(),
				request: make(chan int, 1),
			}
			p.createReply.BufferMaxLength = 4
			p.state.set(running)
			p.request <- 4
			close(p.request)
			var done sync.WaitGroup
			done.Add(1)
			go func() { defer done.Done(); p.run() }()
			close(start)
			for i := 0; i < 10000; i++ {
				_ = p.Error()
			}
			done.Wait()
			expected := result
			if result == EndOfData {
				expected = nil
			}
			if got := p.Error(); got != expected {
				t.Fatalf("Error() = %v, want %v", got, expected)
			}
			if p.Running() {
				t.Fatal("reader result did not stop playback")
			}
		})
	}
}

// Start still clears the previous reader error. Replies are ordinary empty
// protocol acknowledgments; no reader callback or error lock spans protocol I/O.
func TestPlaybackErrorResetOnStart(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	var protocol proto.Client
	protocol.SetTimeout(time.Second)
	protocol.Open(clientConn)
	done := make(chan error, 1)
	go func() {
		for i := 0; i < 2; i++ {
			header := make([]byte, 20)
			if _, err := io.ReadFull(serverConn, header); err != nil {
				done <- err
				return
			}
			payload := make([]byte, binary.BigEndian.Uint32(header[:4]))
			if _, err := io.ReadFull(serverConn, payload); err != nil {
				done <- err
				return
			}
			response := make([]byte, 30)
			binary.BigEndian.PutUint32(response[:4], 10)
			binary.BigEndian.PutUint32(response[4:8], 0xffffffff)
			response[20], response[25] = 'L', 'L'
			binary.BigEndian.PutUint32(response[21:25], proto.OpReply)
			copy(response[26:30], payload[6:10])
			if _, err := serverConn.Write(response); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	p := &PlaybackStream{
		c: &Client{c: &protocol}, state: newStateMachine(), err: errors.New("previous reader error"),
		request: make(chan int, 1), started: make(chan bool, 1),
	}
	p.started <- true
	p.createReply.BufferTargetLength = 4
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
			}
		}
	}()
	p.Start()
	close(stopPolling)
	<-pollingDone
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := p.Error(); got != nil {
		t.Fatalf("Error after Start = %v", got)
	}
	if !p.Running() {
		t.Fatal("Start did not set running state")
	}
	if got := <-p.request; got != 4 {
		t.Fatalf("reader request = %d, want4", got)
	}
	var zero PlaybackStream
	if err := zero.Error(); err != nil {
		t.Fatalf("zero value Error = %v", err)
	}
}
