package crontab

import (
	"github.com/jylc/cloudserver/models"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

var Cron *cron.Cron

func Init() {
	logrus.Info("Initialize scheduled tasks")
	options := models.GetSettingByNames(
		"cron_garbage_collect",
		"cron_recycle_upload_session",
	)

	Cron := cron.New()
	for k, v := range options {
		var handler func()
		switch k {
		case "cron_garbage_collect":
			handler = garbageCollect
		case "cron_recycle_upload_session":
			handler = uploadSessionCollect
		default:
			logrus.Warningf("Unknown scheduled task type [%s], skipping", k)
			continue
		}

		if _, err := Cron.AddFunc(v, handler); err != nil {
			logrus.Warningf("Unable to start scheduled task [%s],%s", k, err)
		}
	}
	Cron.Start()
}
