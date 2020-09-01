package air_etcd

import (
	"github.com/airingone/config"
	"github.com/airingone/log"
	"testing"
)

func TestEtcdServer(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	var serverInfo ServerInfoSt
	serverInfo.Name = "server1"
	serverInfo.Ip = "127.0.0.11"
	serverInfo.Port = 8080
	var endpoints []string //etcd集群的地址
	endpoints = append(endpoints, "127.0.0.1:2380")
	etcdClient, err := NewEtcdServer(serverInfo, endpoints)
	if err != nil {
		t.Logf("errorf:%+v", err)
	}

	etcdClient.Start()
	defer etcdClient.Stop()
	select {}
}

func TestEtcdClient(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	var endpoints []string //etcd集群的地址
	endpoints = append(endpoints, "127.0.0.1:2380")
	var serverNames []string
	serverNames = append(serverNames, "server1")
	serverNames = append(serverNames, "server2")
	etcdClient, err := NewEtcdClient(serverNames, endpoints)
	if err != nil {
		t.Logf("errorf:%+v", err)
	}

	etcdClient.Watcher()
	defer etcdClient.Stop()
	select {}
}
