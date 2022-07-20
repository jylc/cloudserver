package task

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/conf"
	"github.com/sirupsen/logrus"
)

var TaskPool Pool

type Pool interface {
	Add(num int)
	Submit(job Job)
}

type AsyncPool struct {
	idleWorker chan int
}

func (pool *AsyncPool) Add(num int) {
	for i := 0; i < num; i++ {
		pool.idleWorker <- 1
	}
}

func (pool *AsyncPool) obtainWorker() Worker {
	select {
	case <-pool.idleWorker:
		return &GeneralWorker{}
	}
}

func (pool *AsyncPool) freeWorker() {
	pool.Add(1)
}

func (pool *AsyncPool) Submit(job Job) {
	go func() {
		logrus.Debugf("Waiting to get worker")
		worker := pool.obtainWorker()
		logrus.Debugf("Get worker")
		worker.Do(job)
		logrus.Debugf("Release worker")
		pool.freeWorker()
	}()
}

func Init() {
	maxWorker := models.GetIntSetting("max_worker_num", 10)
	TaskPool = &AsyncPool{
		idleWorker: make(chan int, maxWorker),
	}
	TaskPool.Add(maxWorker)
	logrus.Infof("Initialize task queue, workernum =%d", maxWorker)
	if conf.Sc.Role == "master" {
		Resume(TaskPool)
	}
}
