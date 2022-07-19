package onedrive

import "sync"

type CredentialLock interface {
	Lock(uint)
	Unlock(uint)
}

var GlobalMutex = mutexMap{}

type mutexMap struct {
	locks sync.Map
}

func (m *mutexMap) Lock(id uint) {
	lock, _ := m.locks.LoadOrStore(id, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
}

func (m *mutexMap) Unlock(id uint) {
	lock, _ := m.locks.LoadOrStore(id, &sync.Mutex{})
	lock.(*sync.Mutex).Unlock()
}
