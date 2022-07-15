package backoff

import "time"

type Backoff interface {
	Next() bool
	Reset()
}

type ConstantBackoff struct {
	Sleep time.Duration
	Max   int
	tried int
}

func (c *ConstantBackoff) Next() bool {
	c.tried++
	if c.tried > c.Max {
		return false
	}
	time.Sleep(c.Sleep)
	return true
}

func (c *ConstantBackoff) Reset() {
	c.tried = 0
}
