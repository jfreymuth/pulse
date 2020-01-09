package pulse

import (
	"errors"

	"github.com/jfreymuth/pulse/proto"
)

const volumeHundredPercent = 65536

func ratioToVolume(r float32) (uint32, error) {
	vf := r * volumeHundredPercent
	if vf < 0 || vf > 0xFFFFFFFF {
		return 0, errors.New("volume out of range")
	}
	return uint32(vf), nil
}

// SetSinkVolume sets volume of the chnnels.
// 1.0 means maximum volume and the sink may support software boosted value larger than 1.0.
// Number of the arguments should be matched to the number of the channels.
// If only one argument is given, volume of all channels will be set to it.
func (c *Client) SetSinkVolume(s *Sink, volume ...float32) error {
	var cvol proto.ChannelVolumes
	switch len(volume) {
	case 1:
		v, err := ratioToVolume(volume[0])
		if err != nil {
			return err
		}
		for range s.info.ChannelVolumes {
			cvol = append(cvol, v)
		}
	case len(s.info.ChannelVolumes):
		for _, vRatio := range volume {
			v, err := ratioToVolume(vRatio)
			if err != nil {
				return err
			}
			cvol = append(cvol, v)
		}
	default:
		return errors.New("invalid volume length")
	}
	return c.c.Request(&proto.SetSinkVolume{
		SinkIndex:      s.info.SinkIndex,
		ChannelVolumes: cvol,
	}, &proto.SetSinkVolumeReply{})
}

// SetSourceVolume sets volume of the chnnels.
// 1.0 means maximum volume and the source may support software boosted value larger than 1.0.
// Number of the arguments should be matched to the number of the channels.
// If only one argument is given, volume of all channels will be set to it.
func (c *Client) SetSourceVolume(s *Source, volume ...float32) error {
	var cvol proto.ChannelVolumes
	switch len(volume) {
	case 1:
		v, err := ratioToVolume(volume[0])
		if err != nil {
			return err
		}
		for range s.info.ChannelVolumes {
			cvol = append(cvol, v)
		}
	case len(s.info.ChannelVolumes):
		for _, vRatio := range volume {
			v, err := ratioToVolume(vRatio)
			if err != nil {
				return err
			}
			cvol = append(cvol, v)
		}
	default:
		return errors.New("invalid volume length")
	}
	return c.c.Request(&proto.SetSourceVolume{
		SourceIndex:    s.info.SourceIndex,
		ChannelVolumes: cvol,
	}, &proto.SetSourceVolumeReply{})
}
