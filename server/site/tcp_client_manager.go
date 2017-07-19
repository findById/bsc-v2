package site

import "sync"

type ClientManager struct {
	ConnMap map[string]*TcpClient
	Lock    sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		ConnMap:make(map[string]*TcpClient),
	}
}

func (this *ClientManager) Add(client *TcpClient) {
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
	this.Lock.Lock()
	// defer this.Lock.Unlock();
	c, ok := this.ConnMap[id]
	if ok {
		c.Close()
		delete(this.ConnMap, id)
	}
	this.Lock.Unlock()
}

func (this *ClientManager) GetTcpClientByChannelId(id uint64, cId string) *TcpClient {
	for _, c := range this.CloneMap() {
		if c != nil && !c.IsClosed && c.ChannelId == id && c.ClientId == cId {
			return c
		}
	}
	return nil
}

func (this *ClientManager) GetTcpClientByClientId(cId string) *TcpClient {
	for _, c := range this.CloneMap() {
		if c != nil && !c.IsClosed && c.ClientId == cId {
			return c
		}
	}
	return nil
}

func (this *ClientManager) GetClient(id string) *TcpClient {
	this.Lock.RLock()
	defer this.Lock.RUnlock()
	return this.ConnMap[id]
}

func (this *ClientManager) Size() int {
	return len(this.ConnMap)
}

func (this *ClientManager) CloneMap() []*TcpClient {
	this.Lock.RLock()
	// defer this.Lock.RUnlock();
	closedIds := make([]string, 0)

	clone := make([]*TcpClient, len(this.ConnMap))
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
