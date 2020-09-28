package air_etcd

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"go.etcd.io/etcd/clientv3"
	"math/rand"
	"strings"
	"sync"
	"time"
)

//etcd普通client，用于从etcd集群获取服务地址

var AllEtcdClients map[string]*EtcdClient //全局etcd client
var AllEtcdClientsRmu sync.RWMutex

//用户请求端获取地址etcd
type EtcdClient struct {
	Ctx            context.Context  //context
	Client         *clientv3.Client //etcd client
	ServerName     string           //服务名
	ServerInfos    []ServerInfoSt   //服务信息
	ServerInfoLock sync.RWMutex
}

//程序启动时为每一个etcd client初始化etcd
//config: etcd集群配置
//addrs: 需要获取配置的服务名
func InitEtcdClient(config config.ConfigEtcd, addrs ...string) {
	if AllEtcdClients == nil {
		AllEtcdClients = make(map[string]*EtcdClient)
	}

	for _, addr := range addrs {
		index := strings.IndexAny(addr, ":")
		if index == -1 {
			return
		}
		addrType := addr[0:index]
		serverName := addr[index+1:]

		if addrType == "etcd" { //校验
			client, err := NewEtcdClient(serverName, config.Addrs)
			if err != nil {
				log.Error("[ETCD]: NewEtcdClient err: %+v", err)
				return
			}
			AllEtcdClientsRmu.Lock()
			if oldClient, ok := AllEtcdClients[serverName]; ok { //已初始化过则关闭
				oldClient.Stop()
			}
			AllEtcdClients[serverName] = client
			AllEtcdClientsRmu.Unlock()
		}
	}
}

//close all etcd client
func CloseEtcdClient() {
	if AllEtcdClients == nil {
		return
	}
	for _, cli := range AllEtcdClients {
		if cli != nil {
			cli.Stop()
		}
	}
}

//根据服务名拉取已初始化过的etcd client
//serverNames: 服务名
func GetEtcdClientByServerName(serverNames string) (*EtcdClient, error) {
	if AllEtcdClients == nil {
		return nil, errors.New("etcd all client not init")
	}
	AllEtcdClientsRmu.RLock()
	defer AllEtcdClientsRmu.RUnlock()
	if _, ok := AllEtcdClients[serverNames]; !ok {
		return nil, errors.New("etcd client not init")
	}

	return AllEtcdClients[serverNames], nil
}

//创建client对象
//serverName: 服务名
//endpoints: etcd集群地址
func NewEtcdClient(serverName string, endpoints []string) (*EtcdClient, error) {
	cli, err := NewEtcd(endpoints)
	if err != nil {
		return nil, err
	}
	client := &EtcdClient{
		Ctx:        context.Background(),
		Client:     cli,
		ServerName: serverName,
	}
	client.Watcher()

	return client, err
}

//监控以服务名为前缀的key
func (c *EtcdClient) Watcher() {
	log.Info("[ETCD]: etcd client Watcher severName: %s", c.ServerName)
	go c.watcher(c.ServerName)
}

//监控服务注册变化
//serverName: 服务名
func (c *EtcdClient) watcher(serverName string) error {
	defer func() {
		if r := recover(); r != nil {
			log.PanicTrack()
		}
	}()

	resp, err := c.Client.Get(context.Background(), c.ServerName, clientv3.WithPrefix())
	if err == nil {
		for _, kv := range resp.Kvs {
			var serverInfo ServerInfoSt
			err := json.Unmarshal([]byte(kv.Value), &serverInfo)
			if err == nil {
				c.insertServerInfo(string(kv.Key), serverInfo)
			}
		}
	}
	defer c.Stop()

	rCh := c.Client.Watch(c.Ctx, serverName, clientv3.WithPrefix())
	for r := range rCh { //会一直等待
		for _, ev := range r.Events {
			log.Info("[ETCD]: event type:%d, key:%s, value:%s", ev.Type, string(ev.Kv.Key), string(ev.Kv.Value))
			switch ev.Type {
			case clientv3.EventTypePut:
				var serverInfo ServerInfoSt
				err := json.Unmarshal(ev.Kv.Value, &serverInfo)
				if err != nil {
					log.Error("[ETCD]: EtcdClient Unmarshal ServerInfo err")
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

//将key信息写入到ServerInfos
//key: etcd数据key
//serverInfo: serverc信息
func (c *EtcdClient) insertServerInfo(key string, serverInfo ServerInfoSt) {
	c.ServerInfoLock.Lock()
	defer c.ServerInfoLock.Unlock()
	c.ServerInfos = append(c.ServerInfos, serverInfo)
}

//将key信息移除ServerInfos
//key: etcd数据key
func (c *EtcdClient) deleteServerInfo(key string) {
	c.ServerInfoLock.Lock()
	defer c.ServerInfoLock.Unlock()
	_, ip := GetKeyInfo(key)
	var newInfo []ServerInfoSt
	for i, _ := range c.ServerInfos {
		if c.ServerInfos[i].Ip == ip {
			continue
		}
		newInfo = append(newInfo, c.ServerInfos[i])
	}
	c.ServerInfos = newInfo
}

//拉取全量服务地址
func (c *EtcdClient) GetAllServerAddr() ([]ServerInfoSt, error) {
	c.ServerInfoLock.RLock()
	defer c.ServerInfoLock.RUnlock()
	var infos []ServerInfoSt
	for _, v := range c.ServerInfos {
		infos = append(infos, v)
	}

	return infos, nil
}

//随机负载均衡拉取一个地址
func (c *EtcdClient) RandGetServerAddr() (ServerInfoSt, error) {
	c.ServerInfoLock.RLock()
	defer c.ServerInfoLock.RUnlock()
	var info ServerInfoSt
	size := len(c.ServerInfos)
	if size < 1 {
		return info, errors.New("etcd addr not exist")
	}
	rand.Seed(time.Now().UnixNano())
	index := rand.Int()
	info = c.ServerInfos[index%size]

	return info, nil
}
