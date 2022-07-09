package email

import (
	"github.com/jylc/cloudserver/models"
	"github.com/sirupsen/logrus"
	"sync"
)

var Client Driver
var Lock sync.RWMutex

func Init() {
	logrus.Println("init email queue")
	Lock.Lock()
	defer Lock.Unlock()

	if Client != nil {
		Client.Close()
	}

	options := models.GetSettingByNames("fromName",
		"fromAdress",
		"smtpHost",
		"replyTo",
		"smtpUser",
		"smtpPass",
		"smtpEncryption")
	port := models.GetIntSetting("smtpPort", 25)
	keepAlive := models.GetIntSetting("mail_keepalive", 30)
	client := NewSMTPClient(SMTPConfig{
		Name:       options["fromName"],
		Address:    options["fromAdress"],
		ReplyTo:    options["replyTo"],
		Host:       options["smtpHost"],
		Port:       port,
		User:       options["smtpUser"],
		Password:   options["smtpPass"],
		Keepalive:  keepAlive,
		Encryption: models.IsTrueVal(options["smtpEncryption"]),
	})
	Client = client
}
