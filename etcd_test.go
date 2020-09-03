package air_etcd

import (
	"context"
	"fmt"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
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
	t.Logf("addrs: %+v", etcdClient.GetServerInfo("server1"))

	select {}
}

//grpc测试 todo
func TestGrpcClient(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	var endpoints []string //etcd集群的地址
	endpoints = append(endpoints, "127.0.0.1:2380")
	r := NewGrpcResolver("server1", endpoints)
	resolver.Register(r)

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	addr := fmt.Sprintf("%s:///%s", r.Scheme(), "g.srv.mail")
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(),
		// grpc.WithBalancerName(roundrobin.Name),
		//指定初始化round_robin => balancer (后续可以自行定制balancer和 register、resolver 同样的方式)
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithBlock())
	defer conn.Close()
	if err != nil {
	}

}
