package air_etcd

import (
	"github.com/airingone/config"
	"github.com/airingone/log"
	"testing"
	"time"
)

//服务端注册
func TestEtcdServer(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化
	/*
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
		defer etcdClient.Stop()*/

	var endpoints []string //etcd集群的地址
	endpoints = append(endpoints, "127.0.0.1:2380")
	RegisterLocalServerToEtcd("server1", 8080, endpoints)
	select {}
}

//普通拉取服务地址client
func TestEtcdClient(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	var endpoints []string //etcd集群的地址
	endpoints = append(endpoints, "127.0.0.1:2380")
	etcdClient, err := NewEtcdClient("server1", endpoints)
	if err != nil {
		t.Logf("errorf:%+v", err)
	}

	time.Sleep(5 * time.Second)
	addr, err := etcdClient.GetAllServerAddr()
	log.Error("[ETCD]: addrs1 %+v", addr)

	cli, err := GetEtcdClientByServerName("server1")
	if err == nil {
		addr, _ := cli.RandGetServerAddr()
		log.Error("[ETCD]: addrs2 %+v", addr)
	}

	select {}
}

//grpc测试
func TestGrpcClient(t *testing.T) {
	//goto https://github.com/airingone/air-grpc/blob/master/grpc_test.go
}
