package onedrive

import "time"

type Option interface {
	apply(*options)
}

type options struct {
	redirect          string
	code              string
	refreshToken      string
	conflictBehavior  string
	expires           time.Time
	useDriverResource bool
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func WithCode(t string) Option {
	return optionFunc(func(o *options) {
		o.code = t
	})
}

func WithRefreshToken(t string) Option {
	return optionFunc(func(o *options) {
		o.refreshToken = t
	})
}

func WithConflictBehavior(t string) Option {
	return optionFunc(func(o *options) {
		o.conflictBehavior = t
	})
}
func WithDriverResource(t bool) Option {
	return optionFunc(func(o *options) {
		o.useDriverResource = t
	})
}

func newDefaultOption() *options {
	return &options{
		conflictBehavior:  "fail",
		useDriverResource: true,
		expires:           time.Now().UTC().Add(time.Duration(1) * time.Hour),
	}
}
