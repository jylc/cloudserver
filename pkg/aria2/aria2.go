package aria2

import (
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/balancer"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/mq"
	"sync"
)

var Instance common.Aria2 = &common.DummyAria2{}
var LB balancer.Balancer
var Lock sync.RWMutex

func Init(isReload bool, pool cluster.Pool, mqClient mq.MQ) {
	Lock.Lock()
	LB = balancer.NewBalancer("RoundRobin")
	Lock.Unlock()

	if !isReload {
		unfinished := models.GetDownloadsByStatus(common.Ready, common.Paused, common.Downloading)
		for i := 0; i < len(unfinished); i++ {
			monitor.NewMonitor(&unfinished[i], pool, mqClient)
		}
	}
}
