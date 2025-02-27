package pulse

import "sync"

type streamState int

const (
	idle streamState = iota
	running
	paused
	closed
	serverLost
)

type stateMachine struct {
	state streamState

	lock *sync.RWMutex
}

func newStateMachine() *stateMachine {
	return &stateMachine{
		state: idle,
		lock:  &sync.RWMutex{},
	}
}

func (s *stateMachine) set(state streamState) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.state = state
}

func (s *stateMachine) get() streamState {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.state
}

func (s *stateMachine) is(states ...streamState) bool {
	current := s.get()
	for _, state := range states {
		if current == state {
			return true
		}
	}
	return false
}
