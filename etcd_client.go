package air_etcd

import (
	"context"
	"encoding/json"
	"github.com/airingone/log"
	"go.etcd.io/etcd/clientv3"
	"sync"
)

//用户请求端获取地址etcd
type EtcdClient struct {
	Ctx            context.Context
	Client         *clientv3.Client
	ServerNames    string
	ServerInfos    map[string]ServerInfoSt
	ServerInfoLock sync.RWMutex
}

//创建client对象
func NewEtcdClient(serverNames string, endpoints []string) (*EtcdClient, error) {
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
	go c.watcher(c.ServerNames)

}

//监控服务注册变化
func (c *EtcdClient) watcher(serverName string) error {
	defer func() {
		if r := recover(); r != nil {
			log.PanicTrack()
		}
	}()

	resp, err := c.Client.Get(context.Background(), c.ServerNames, clientv3.WithPrefix())
	if err == nil {
		for _, kv := range resp.Kvs {
			var serverInfo ServerInfoSt
			err := json.Unmarshal([]byte(kv.Value), &serverInfo)
			if err == nil {
				c.insertServerInfo(string(kv.Key), serverInfo)
			}
		}
	}

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

//停止
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

//拉取全量服务地址
func (c *EtcdClient) GetServerInfo(key string) ([]ServerInfoSt, error) {
	c.ServerInfoLock.RLock()
	defer c.ServerInfoLock.RUnlock()
	var infos []ServerInfoSt
	for _, v := range c.ServerInfos {
		infos = append(infos, v)
	}

	return infos, nil
}
