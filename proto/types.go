package proto

import "math"

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

const (
	// Muted (minimal valid) volume (0%, -inf dB)
	VolumeMuted Volume = 0
	// Normal volume (100%, 0 dB)
	VolumeNorm Volume = 0x10000
	// Maximum valid volume we can store.
	VolumeMax Volume = math.MaxUint32 / 2
	// Special 'invalid' volume.
	VolumeInvalid Volume = math.MaxUint32
)

type FormatInfo struct {
	Encoding   byte
	Properties PropList
}

type SubscriptionMask uint32

const (
	SubscriptionMaskSink SubscriptionMask = 1 << iota
	SubscriptionMaskSource
	SubscriptionMaskSinkInput
	SubscriptionMaskSourceInput
	SubscriptionMaskModule
	SubscriptionMaskClient
	SubscriptionMaskSampleCache
	SubscriptionMaskServer
	SubscriptionMaskAutoload
	SubscriptionMaskCard

	SubscriptionMaskNull SubscriptionMask = 0
	SubscriptionMaskAll  SubscriptionMask = 0x02ff
)

type SubscriptionEventType uint32

const (
	EventSink SubscriptionEventType = iota
	EventSource
	EventSinkSinkInput
	EventSinkSourceOutput
	EventModule
	EventClient
	EventSampleCache
	EventServer
	EventAutoload
	EventCard
	EventFacilityMask SubscriptionEventType = 0xf

	EventNew      SubscriptionEventType = 0x0000
	EventChange   SubscriptionEventType = 0x0010
	EventRemove   SubscriptionEventType = 0x0020
	EventTypeMask SubscriptionEventType = 0x0030
)

func (e SubscriptionEventType) GetFacility() SubscriptionEventType {
	return e & EventFacilityMask
}

func (e SubscriptionEventType) GetType() SubscriptionEventType {
	return e & EventTypeMask
}

func (e SubscriptionEventType) String() string {
	var res string
	switch e.GetType() {
	case EventNew:
		res += "new"
	case EventChange:
		res += "change"
	case EventRemove:
		res += "remove"
	default:
		return "<invalid type>"
	}
	res += " "
	switch e.GetFacility() {
	case EventSink:
		res += "sink"
	case EventSource:
		res += "source"
	case EventSinkSinkInput:
		res += "sink input"
	case EventSinkSourceOutput:
		res += "source output"
	case EventModule:
		res += "module"
	case EventClient:
		res += "client"
	case EventSampleCache:
		res += "sample cache"
	case EventServer:
		res += "server"
	case EventAutoload:
		res += "autoload"
	case EventCard:
		res += "card"
	default:
		return "<invalid facility>"
	}
	return res
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
