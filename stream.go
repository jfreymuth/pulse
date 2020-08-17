package pulse

type streamState int

const (
	idle streamState = iota
	running
	paused
	closed
	serverLost
)
