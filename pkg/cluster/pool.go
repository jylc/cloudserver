package cluster

import (
	"github.com/jylc/cloudserver/models"
	"github.com/jylc/cloudserver/pkg/balancer"
	"github.com/sirupsen/logrus"
	"sync"
)

var Default *NodePool

var featureGroup = []string{"aria2"}

type Pool interface {
	BalanceNodeByFeature(feature string, lb balancer.Balancer) (error, Node)

	GetNodeByID(id uint) Node

	Add(node *models.Node)

	Delete(id uint)
}

type NodePool struct {
	active     map[uint]Node
	inactive   map[uint]Node
	featureMap map[string][]Node
	lock       sync.RWMutex
}

func (pool *NodePool) BalanceNodeByFeature(feature string, lb balancer.Balancer) (error, Node) {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	if nodes, ok := pool.featureMap[feature]; ok {
		err, res := lb.NextPeer(nodes)
		if err == nil {
			return nil, res.(Node)
		}
		return err, nil
	}
	return ErrFeatureNotExist, nil
}

func (pool *NodePool) GetNodeByID(id uint) Node {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if node, ok := pool.active[id]; ok {
		return node
	}
	return pool.inactive[id]
}

func Init() {
	Default = &NodePool{}
	Default.Init()
	if err := Default.initFromDB(); err != nil {
		logrus.Warningf("node pool initialization failed, %s", err)
	}
}

func (pool *NodePool) Init() {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	pool.featureMap = make(map[string][]Node)
	pool.active = make(map[uint]Node)
	pool.inactive = make(map[uint]Node)
}

func (pool *NodePool) initFromDB() error {
	nodes, err := models.GetNodesByStatus(models.NodeActive)
	if err != nil {
		return err
	}

	pool.lock.Lock()
	for i := 0; i < len(nodes); i++ {
		pool.add(&nodes[i])
	}
	pool.lock.Unlock()
	pool.buildIndexMap()
	return nil
}

func (pool *NodePool) add(node *models.Node) {
	newNode := NewNodeFromDBModel(node)
	if newNode.IsActive() {
		pool.active[node.ID] = newNode
	} else {
		pool.inactive[node.ID] = newNode
	}
	newNode.SubscribeStatusChange(func(isActive bool, id uint) {
		pool.nodeStatusChange(isActive, id)
	})
}

func (pool *NodePool) buildIndexMap() {
	pool.lock.Lock()

	for _, feature := range featureGroup {
		pool.featureMap[feature] = make([]Node, 0)
	}

	for _, v := range pool.active {
		for _, feature := range featureGroup {
			if v.IsFeatureEnabled(feature) {
				pool.featureMap[feature] = append(pool.featureMap[feature], v)
			}
		}
	}
	pool.lock.Unlock()
}

func (pool *NodePool) nodeStatusChange(isActive bool, id uint) {
	logrus.Debugf("slave [ID=%d] status changed [Active=%t]", id, isActive)
	var node Node
	pool.lock.Lock()
	if n, ok := pool.inactive[id]; ok {
		node = n
		delete(pool.inactive, id)
	} else {
		node = pool.active[id]
		delete(pool.active, id)
	}

	if isActive {
		pool.active[id] = node
	} else {
		pool.inactive[id] = node
	}

	pool.lock.Unlock()
	pool.buildIndexMap()
}

func (pool *NodePool) Add(node *models.Node) {
	pool.lock.Lock()
	defer pool.buildIndexMap()
	defer pool.lock.Unlock()

	var (
		old Node
		ok  bool
	)

	if old, ok = pool.active[node.ID]; !ok {
		old, ok = pool.inactive[node.ID]
	}

	if old != nil {
		go old.Init(node)
		return
	}
	pool.add(node)
}

func (pool *NodePool) Delete(id uint) {
	pool.lock.Lock()
	defer pool.buildIndexMap()
	defer pool.lock.Unlock()

	if node, ok := pool.active[id]; ok {
		node.Kill()
		delete(pool.active, id)
		return
	}

	if node, ok := pool.inactive[id]; ok {
		node.Kill()
		delete(pool.inactive, id)
		return
	}
}
