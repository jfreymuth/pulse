package proto

type Error uint32

const (
	ok Error = iota
	ErrAccessDenied
	ErrUnknownCommand
	ErrInvalidArgument
	ErrEntityExists
	ErrNoSuchEntity
	ErrConnectionRefused
	ErrProtocolError
	ErrTimeout
	ErrNoAuthenticationKey
	ErrInternalError
	ErrConnectionTerminated
	ErrEntityKilled
	ErrInvalidServer
	ErrModuleInitializationFailed
	ErrBadState
	ErrNoData
	ErrIncompatibleProtocolVersion
	ErrTooLarge
	ErrNotSupported
	ErrUnknownErrorCode
	ErrNoSuchExtension
	ErrObsoleteFunctionality
	ErrMissingImplementation
	ErrClientForked
	ErrInputOutputError
	ErrDeviceOrEesourceBusy
)

func (e Error) Error() string {
	switch e {
	case ok:
		return "pulseaudio: ok"
	case ErrAccessDenied:
		return "pulseaudio: access denied"
	case ErrUnknownCommand:
		return "pulseaudio: unknown command"
	case ErrInvalidArgument:
		return "pulseaudio: invalid argument"
	case ErrEntityExists:
		return "pulseaudio: entity exists"
	case ErrNoSuchEntity:
		return "pulseaudio: no such entity"
	case ErrConnectionRefused:
		return "pulseaudio: connection refused"
	case ErrProtocolError:
		return "pulseaudio: protocol error"
	case ErrTimeout:
		return "pulseaudio: timeout"
	case ErrNoAuthenticationKey:
		return "pulseaudio: no authentication key"
	case ErrInternalError:
		return "pulseaudio: internal error"
	case ErrConnectionTerminated:
		return "pulseaudio: connection terminated"
	case ErrEntityKilled:
		return "pulseaudio: entity killed"
	case ErrInvalidServer:
		return "pulseaudio: invalid server"
	case ErrModuleInitializationFailed:
		return "pulseaudio: module initialization failed"
	case ErrBadState:
		return "pulseaudio: bad state"
	case ErrNoData:
		return "pulseaudio: no data"
	case ErrIncompatibleProtocolVersion:
		return "pulseaudio: incompatible protocol version"
	case ErrTooLarge:
		return "pulseaudio: too large"
	case ErrNotSupported:
		return "pulseaudio: not supported"
	case ErrUnknownErrorCode:
		return "pulseaudio: unknown error code"
	case ErrNoSuchExtension:
		return "pulseaudio: no such extension"
	case ErrObsoleteFunctionality:
		return "pulseaudio: obsolete functionality"
	case ErrMissingImplementation:
		return "pulseaudio: missing implementation"
	case ErrClientForked:
		return "pulseaudio: client forked"
	case ErrInputOutputError:
		return "pulseaudio: input/output error"
	case ErrDeviceOrEesourceBusy:
		return "pulseaudio: device or resource busy"
	}
	return "pulseaudio: invalid error code"
}
