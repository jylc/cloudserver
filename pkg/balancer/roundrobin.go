package balancer

import (
	"reflect"
	"sync/atomic"
)

type RoundRobin struct {
	current uint64
}

func (r *RoundRobin) NextPeer(nodes interface{}) (error, interface{}) {
	v := reflect.ValueOf(nodes)
	if v.Kind() != reflect.Slice {
		return ErrInputNotSlice, nil
	}

	if v.Len() == 0 {
		return ErrNoAvailableNode, nil
	}
	next := r.NextIndex(v.Len())
	return nil, v.Index(next).Interface()
}

func (r *RoundRobin) NextIndex(total int) int {
	return int(atomic.AddUint64(&r.current, uint64(1)) % uint64(total))
}
