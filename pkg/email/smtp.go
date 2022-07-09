package email

import (
	mail2 "github.com/go-mail/mail"
	"github.com/sirupsen/logrus"
	"time"
)

type SMTP struct {
	Config SMTPConfig
	ch     chan *mail2.Message
	chOpen bool
}

type SMTPConfig struct {
	Name       string
	Address    string
	ReplyTo    string
	Host       string
	Port       int
	User       string
	Password   string
	Encryption bool
	Keepalive  int
}

func NewSMTPClient(config SMTPConfig) *SMTP {
	client := &SMTP{
		Config: config,
		ch:     make(chan *mail2.Message, 30),
		chOpen: false,
	}

	client.Init()
	return client
}

func (client *SMTP) Send(to, title, body string) error {
	if !client.chOpen {
		return ErrChanNotOpen
	}
	m := mail2.NewMessage()
	m.SetAddressHeader("From", client.Config.Address, client.Config.Name)
	m.SetAddressHeader("Reply-To", client.Config.ReplyTo, client.Config.Name)
	m.SetHeader("To", to)
	m.SetHeader("Subject", title)
	m.SetBody("text/html", body)
	client.ch <- m
	return nil
}

func (client *SMTP) Close() {
	if client.ch != nil {
		close(client.ch)
	}
}

func (client *SMTP) Init() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				client.chOpen = false
				logrus.Errorf("mail queue has some error, %s , 10's later reset\n", err)
				time.Sleep(time.Duration(10) * time.Second)
				client.Init()
			}
		}()

		d := mail2.NewDialer(client.Config.Host, client.Config.Port, client.Config.User, client.Config.Password)
		d.Timeout = time.Duration(client.Config.Keepalive*5) * time.Second
		client.chOpen = true
		d.SSL = false
		if client.Config.Encryption {
			d.SSL = true
		}
		d.StartTLSPolicy = mail2.OpportunisticStartTLS

		var s mail2.SendCloser
		var err error
		open := false
		for {
			select {
			case m, ok := <-client.ch:
				if !ok {
					logrus.Debug("mail queue closed")
					client.chOpen = false
					return
				}
				if !open {
					if s, err = d.Dial(); err != nil {
						panic(err)
					}
					open = true
				}
				if err := mail2.Send(s, m); err != nil {
					logrus.Warningf("mail send failed, %s\n", err)
				} else {
					logrus.Debug("mail has send")
				}

			case <-time.After(time.Duration(client.Config.Keepalive) * time.Second):
				if open {
					if err := s.Close(); err != nil {
						logrus.Warningf("cannot close smtp connction, %s\n", err)
					}
					open = false
				}
			}
		}
	}()
}
