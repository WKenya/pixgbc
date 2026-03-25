package core

import "errors"

var (
	ErrUnsupportedFormat = errors.New("unsupported image format")
	ErrImageTooLarge     = errors.New("image exceeds configured limits")
	ErrInvalidConfig     = errors.New("invalid render config")
	ErrUnknownPalette    = errors.New("unknown palette preset")
	ErrUnknownMode       = errors.New("unknown render mode")
	ErrUnsupportedAlpha  = errors.New("unsupported alpha configuration")
	ErrReviewNotFound    = errors.New("review artifact not found")
	ErrNotImplemented    = errors.New("feature not implemented")
)
