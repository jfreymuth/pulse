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

type ChannelVolumes []Volume

// Avg returns the average volume of all the channels.
func (cv ChannelVolumes) Avg() Volume {
	// This matches pa_cvolume_avg.
	avg := uint64(0)
	for _, vol := range cv {
		avg += uint64(vol)
	}
	return Volume(avg / uint64(len(cv)))
}

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

// Convert a linear volume to a Volume value. The input volume must have a range
// from 0.0 to 1.0, values outside this range will be clamped.
func LinearVolume(v float64) Volume {
	// This is the formula as pa_sw_volume_from_linear in PulseAudio.

	return NormVolume(math.Cbrt(v))
}

// Convert a normalized volume value (0.0..1.0 for 0%..100%) to a Volume value.
// Volume values that are out of range will be clipped.
func NormVolume(v float64) Volume {
	rawVolume := int64(math.Round(v * float64(VolumeNorm)))
	if rawVolume < int64(VolumeMuted) {
		rawVolume = 0
	} else if rawVolume > int64(VolumeMax) {
		rawVolume = int64(VolumeMax)
	}
	return Volume(rawVolume)
}

// Return the linear volume, from 0% to 100% as a value from 0.0..1.0.
func (v Volume) Linear() float64 {
	// This is the same formula as pa_sw_volume_to_linear in PulseAudio.
	f := v.Norm()
	return f * f * f
}

// Convert a Volume value back to a normalized volume, where 1.0 means 100%.
func (v Volume) Norm() float64 {
	if v > VolumeMax {
		return 0
	}

	if v <= VolumeMuted {
		return 0.0
	}

	if v == VolumeNorm {
		return 1.0
	}

	return (float64(v) / float64(VolumeNorm))
}

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
