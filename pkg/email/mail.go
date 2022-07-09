package email

import (
	"errors"
	"strings"
)

var (
	ErrChanNotOpen      = errors.New("mail queue is not open")
	ErrNoActivateDriver = errors.New("no mail sending service available")
)

type Driver interface {
	Close()
	Send(to, title, body string) error
}

func Send(to, title, body string) error {
	if strings.HasSuffix(to, "@login.qq.com") {
		return nil
	}
	Lock.RLock()
	defer Lock.RUnlock()
	if Client == nil {
		return ErrNoActivateDriver
	}
	return Client.Send(to, title, body)
}
