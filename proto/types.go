package proto

const Undefined = 0xFFFFFFFF

const (
	FormatUint8     = 0
	FormatInt16LE   = 3
	FormatInt16BE   = 4
	FormatFloat32LE = 5
	FormatFloat32BE = 6
	FormatInt32LE   = 7
	FormatInt32BE   = 8
)

const (
	ChannelMono           = 0
	ChannelLeft           = 1
	ChannelRight          = 2
	ChannelCenter         = 3
	ChannelFrontLeft      = 1
	ChannelFrontRight     = 2
	ChannelFrontCenter    = 3
	ChannelRearCenter     = 4
	ChannelRearLeft       = 5
	ChannelRearRight      = 6
	ChannelLFE            = 7
	ChannelLeftCenter     = 8
	ChannelRightCenter    = 9
	ChannelLeftSide       = 10
	ChannelRightSide      = 11
	ChannelAux0           = 12
	ChannelAux31          = 43
	ChannelTopCenter      = 44
	ChannelTopFrontLeft   = 45
	ChannelTopFrontRight  = 46
	ChannelTopFrontCenter = 47
	ChannelTopRearLeft    = 48
	ChannelTopRearRight   = 49
	ChannelTopRearCenter  = 50
)

const (
	EncodingPCM = 1
)

type SampleSpec struct {
	Format   byte
	Channels byte
	Rate     uint32
}

type Microseconds uint64

type ChannelMap []byte

type ChannelVolumes []uint32

type Time struct {
	Seconds      uint32
	Microseconds uint32
}

type Volume uint32

type FormatInfo struct {
	Encoding   byte
	Properties PropList
}

type PropList map[string]PropListEntry

type PropListEntry []byte

func PropListString(s string) PropListEntry {
	e := make(PropListEntry, len(s)+1)
	copy(e, s)
	return e
}
func (e PropListEntry) String() string {
	if len(e) == 0 || e[len(e)-1] != '\x00' {
		return "<not a string>"
	}
	return string(e[:len(e)-1])
}
