package client

import "sync"

type ClientManager struct {
	ConnMap     map[string]*Client
	Lock        sync.RWMutex
}

func NewClientManager() *ClientManager {
	cm := &ClientManager{
		ConnMap: make(map[string]*Client),
	}
	return cm
}

func (this *ClientManager) AddClient(client *Client) {
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

func (this *ClientManager) RemoveClient(id string) {
	if this.Size() <= 1 {
		return
	}
	this.Lock.Lock()
	// defer this.Lock.Unlock();
	c, ok := this.ConnMap[id]
	if ok {
		c.Close()
		delete(this.ConnMap, id)
	}
	this.Lock.Unlock()
}

func (this *ClientManager) GetClient(id string) *Client {
	this.Lock.RLock()
	defer this.Lock.RUnlock()
	return this.ConnMap[id]
}

func (this *ClientManager) Size() int {
	return len(this.ConnMap)
}

func (this *ClientManager) CloneMap() []*Client {
	this.Lock.RLock()
	// defer this.Lock.RUnlock();
	closedIds := make([]string, 0)

	clone := make([]*Client, len(this.ConnMap))
	i := 0
	for _, v := range this.ConnMap {
		if v.IsClosed {
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

func (this *ClientManager) remove(ids []string) {
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
