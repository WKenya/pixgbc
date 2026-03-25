package ioimg

type Limits struct {
	MaxWidth     int
	MaxHeight    int
	MaxPixels    int64
	MaxFileBytes int64
}

func DefaultLimits() Limits {
	return Limits{
		MaxWidth:     4096,
		MaxHeight:    4096,
		MaxPixels:    4096 * 4096,
		MaxFileBytes: 32 << 20,
	}
}
