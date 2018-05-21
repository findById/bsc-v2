package client

import "sync"

type ProxyClientManager struct {
	AuthToken string
	ConnMap   map[string]*ProxyClient
	Lock      sync.RWMutex
}

func NewClientManager() *ProxyClientManager {
	cm := &ProxyClientManager{
		ConnMap: make(map[string]*ProxyClient),
	}
	return cm
}

func (this *ProxyClientManager) AddClient(client *ProxyClient) {
	this.Lock.Lock()
	// defer this.Lock.Unlock();
	c, ok := this.ConnMap[client.Id]
	if ok {
		c.Close()
		delete(this.ConnMap, client.Id)
	}
	this.ConnMap[client.Id] = client
	this.Lock.Unlock()
}

func (this *ProxyClientManager) RemoveClient(id string) {
	this.Lock.Lock()
	// defer this.Lock.Unlock();
	c, ok := this.ConnMap[id]
	if ok {
		c.Close()
		delete(this.ConnMap, id)
	}
	this.Lock.Unlock()
}

func (this *ProxyClientManager) GetClient(id string) *ProxyClient {
	this.Lock.RLock()
	defer this.Lock.RUnlock()
	return this.ConnMap[id]
}

func (this *ProxyClientManager) Size() int {
	return len(this.ConnMap)
}

func (this *ProxyClientManager) CloneMap() []*ProxyClient {
	this.Lock.RLock()
	// defer this.Lock.RUnlock();
	closedIds := make([]string, 0)

	clone := make([]*ProxyClient, len(this.ConnMap))
	i := 0
	for _, v := range this.ConnMap {
		if v == nil || v.IsClosed {
			closedIds = append(closedIds, v.Id)
			continue
		}
		clone[i] = v
		i++
	}
	this.Lock.RUnlock()
	if len(closedIds) > 0 {
		this.remove(closedIds)
	}
	return clone
}

func (this *ProxyClientManager) remove(ids []string) {
	this.Lock.Lock()
	for _, id := range ids {
		c, ok := this.ConnMap[id]
		if ok {
			c.Close()
			delete(this.ConnMap, id)
		}
	}
	this.Lock.Unlock()
}
