package aria2

import (
	"context"
	"fmt"
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/aria2/common"
	"github.com/jylc/cloudserver/pkg/aria2/monitor"
	"github.com/jylc/cloudserver/pkg/aria2/rpc"
	"github.com/jylc/cloudserver/pkg/balancer"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/mq"
	"net/url"
	"sync"
	"time"
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

func TestRPCConnection(server, secret string, timeout int) (rpc.VersionInfo, error) {
	rpcServer, err := url.Parse(server)
	if err != nil {
		return rpc.VersionInfo{}, fmt.Errorf("cannot parse RPC server: %w", err)
	}

	rpcServer.Path = "/jsonrpc"
	caller, err := rpc.New(context.Background(), rpcServer.String(), secret, time.Duration(timeout)*time.Second, nil)
	if err != nil {
		return rpc.VersionInfo{}, fmt.Errorf("cannot initialize rpc connection: %w", err)
	}
	return caller.GetVersion()
}
