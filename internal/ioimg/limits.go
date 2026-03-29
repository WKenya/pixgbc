package ioimg

type Limits struct {
	MaxWidth     int
	MaxHeight    int
	MaxPixels    int64
	MaxFileBytes int64
}

func DefaultLimits() Limits {
	return Limits{
		MaxWidth:     6144,
		MaxHeight:    6144,
		MaxPixels:    28 << 20,
		MaxFileBytes: 32 << 20,
	}
}
