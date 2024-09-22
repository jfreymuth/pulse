package pulse

const (
	idle uint32 = iota
	running
	paused
	closed
	serverLost
)
