package air_etcd

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"strings"
	"time"
)

//etcd基础函数

const (
	EtcdAutoSyncInterval = 60 * time.Second //从etcd集群更新当前集群endpoints，为时间间隔，0表示不同步，
)

//创建etcd client
//endpoints: etcd集群地址
func NewEtcd(endpoints []string) (*clientv3.Client, error) {
	conf := clientv3.Config{
		Endpoints:        endpoints,
		DialTimeout:      5 * time.Second,      //首次连接超时时候，连接上了client自身会维持/重连
		AutoSyncInterval: EtcdAutoSyncInterval, //更新etcd集群的endpoints列表时间间隔，为0则不同步
	}
	cli, err := clientv3.New(conf)
	if err != nil {
		return nil, err
	}

	//检查连接状态，etcd连接超时不提示错误
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = cli.Status(ctx, conf.Endpoints[0])
	if err != nil {
		return nil, err
	}

	return cli, nil
}

//生成数据key
//serverName: 服务名
//ip: 服务ip
func GetHostKey(serverName string, ip string) string {
	return serverName + "/" + ip
}

//根据key解析服务名与ip
//key: etcd数据项key
func GetKeyInfo(key string) (string, string) {
	subStr := strings.Split(key, "/")
	if len(subStr) != 2 {
		return "", ""
	}

	return subStr[0], subStr[1]
}

//todo Service Mesh -- Istio
