package balancer

import "errors"

var (
	ErrInputNotSlice   = errors.New("Input value is not silice")
	ErrNoAvailableNode = errors.New("No nodes avaliable")
)
