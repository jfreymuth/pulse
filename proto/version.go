package proto

type Version uint32

func (v Version) Version() int { return int(v & 0xFFFF) }

func (v Version) Min(u Version) Version {
	flags := v & u & 0xFFFF0000
	v &= 0xFFFF
	if v > u&0xFFFF {
		v = u & 0xFFFF
	}
	return v | flags
}
