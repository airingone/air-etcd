package air_etcd

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/airingone/log"
	"go.etcd.io/etcd/clientv3"
	"sync"
)

//用户请求端获取地址etcd
type EtcdClient struct {
	Ctx            context.Context
	Client         *clientv3.Client
	ServerNames    []string
	ServerInfos    map[string]ServerInfoSt
	ServerInfoLock sync.RWMutex
}

func NewEtcdClient(serverNames []string, endpoints []string) (*EtcdClient, error) {
	cli, err := NewEtcd(endpoints)
	if err != nil {
		return nil, err
	}

	client := &EtcdClient{
		Ctx:         context.Background(),
		Client:      cli,
		ServerNames: serverNames,
		ServerInfos: make(map[string]ServerInfoSt),
	}

	return client, err
}

//监控以服务名为前缀的key
func (c *EtcdClient) Watcher() {
	for _, name := range c.ServerNames {
		go c.watcher(name)
	}
}

func (c *EtcdClient) watcher(serverName string) error {
	defer func() {
		if r := recover(); r != nil {
			log.PanicTrack()
		}
	}()

	rCh := c.Client.Watch(c.Ctx, serverName, clientv3.WithPrefix())
	for r := range rCh { //会一直等待
		for _, ev := range r.Events {
			log.Info("EtcdClient: event type:%d, key:%s, value:%s", ev.Type, string(ev.Kv.Key), string(ev.Kv.Value))
			switch ev.Type {
			case clientv3.EventTypePut:
				var serverInfo ServerInfoSt
				err := json.Unmarshal(ev.Kv.Value, &serverInfo)
				if err != nil {
					log.Error("EtcdClient: Unmarshal ServerInfo err")
				} else {
					c.insertServerInfo(string(ev.Kv.Key), serverInfo)
				}
			case clientv3.EventTypeDelete:
				c.deleteServerInfo(string(ev.Kv.Key))
			}
		}
	}

	return nil
}

func (c *EtcdClient) Stop() {
	c.Client.Close()
}

func (c *EtcdClient) insertServerInfo(key string, serverInfo ServerInfoSt) {
	c.ServerInfoLock.Lock()
	defer c.ServerInfoLock.Unlock()
	c.ServerInfos[key] = serverInfo
}

func (c *EtcdClient) deleteServerInfo(key string) {
	c.ServerInfoLock.Lock()
	defer c.ServerInfoLock.Unlock()
	delete(c.ServerInfos, key)
}

func (c *EtcdClient) GetServerInfo(key string) (ServerInfoSt, error) {
	c.ServerInfoLock.RLock()
	defer c.ServerInfoLock.RUnlock()
	if info, ok := c.ServerInfos[key]; ok {
		return info, nil
	}

	return ServerInfoSt{}, errors.New("not exist")
}
