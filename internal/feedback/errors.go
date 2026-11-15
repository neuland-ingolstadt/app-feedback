package feedback

import "errors"

var (
	ErrRateLimited = errors.New("rate limited")
	ErrInvalid     = errors.New("invalid submission")
)
