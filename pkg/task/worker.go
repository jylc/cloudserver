package task

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

type Worker interface {
	Do(job Job)
}

type GeneralWorker struct {
}

func (worker *GeneralWorker) Do(job Job) {
	logrus.Debugf("Start task")
	job.SetStatus(Processing)

	defer func() {
		if err := recover(); err != nil {
			logrus.Debugf("Task execution error, %s", err)
			job.SetError(&JobError{Msg: "Fatal error", Error: fmt.Sprintf("%s", err)})
			job.SetStatus(Error)
		}
	}()

	job.Do()

	if err := job.GetError(); err != nil {
		logrus.Debugf("Task execution error")
		job.SetStatus(Error)
		return
	}

	logrus.Debugf("Task execution completed")

	job.SetStatus(Complete)
}
