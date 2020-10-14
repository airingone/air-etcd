package air_etcd

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/airingone/log"
	"go.etcd.io/etcd/clientv3"
)

//etcd client，用于上报服务地址到etcd集群

const (
	EtcdServerGrantTtl = 180 //即60s的租约，key 60s后过期，需要续租,EtcdServerGrantTtl为数据过期时间，EtcdServerGrantTtl/3为触发周期
)

//服务信息
type ServerInfoSt struct {
	Name string //当前服务名
	Ip   string //当前服务IP
	Port uint32 //当前服务端口
}

//用于服务端注册的etcd
type EtcdServer struct {
	Ctx        context.Context  //context
	Client     *clientv3.Client //etcd client
	ServerInfo ServerInfoSt     //服务信息
	Key        string           //当前服务的key
	Value      string           //当前服务的value
	LeasdId    clientv3.LeaseID //上一次更新id
	StopState  chan error       //stop
}

//注册服务到etcd
//serverName: 当前服务名
//port: 当前服务端口
//etcdEndpoints: etcd集群地址
func RegisterLocalServerToEtcd(serverName string, port uint32, etcdEndpoints []string) {
	var info ServerInfoSt
	info.Name = serverName
	info.Port = port
	ip, err := GetLoaclIp()
	if err != nil {
		log.Fatal("[ETCD]: RegisterLocalServerToEtcd GetLoaclIp err, err: %+v", err)
		return
	}
	info.Ip = ip

	etcdClient, err := NewEtcdServer(info, etcdEndpoints)
	if err != nil {
		log.Fatal("[ETCD]: RegisterLocalServerToEtcd NewEtcdServer err, err: %+v", err)
		return
	}

	etcdClient.Start()
}

//创建一个etcd client，endpoints为etcd的地址列表
//serverInfo: 服务信息
//endpoints: etcd集群地址
func NewEtcdServer(serverInfo ServerInfoSt, endpoints []string) (*EtcdServer, error) {
	cli, err := NewEtcd(endpoints)
	if err != nil {
		return nil, err
	}

	key := GetHostKey(serverInfo.Name, serverInfo.Ip)
	value, err := json.Marshal(serverInfo)
	if err != nil {
		return nil, err
	}
	client := &EtcdServer{
		Ctx:        context.Background(),
		Client:     cli,
		ServerInfo: serverInfo,
		Key:        key,
		Value:      string(value),
		StopState:  make(chan error),
	}

	return client, err
}

//启动服务注册
func (s *EtcdServer) Start() {
	go s.startProcess()
}

//启动
func (s *EtcdServer) startProcess() error {
	defer func() {
		if r := recover(); r != nil {
			log.PanicTrack()
		}
	}()
	defer func() {
		if s.Client != nil {
			s.Client.Close()
		}
	}()

	ch, err := s.keepAlice()
	if err != nil {
		return err
	}

	for {
		select {
		case err := <-s.StopState:
			log.Info("[ETCD]: Stop")
			return err
		case <-s.Client.Ctx().Done():
			log.Info("[ETCD]: Ctx Done")
			return errors.New("server closed")
		case resp, ok := <-ch:
			if !ok { //租约已关闭
				log.Info("[ETCD]: Revoke, err:%+v", ok)
				return s.revoke()
			}
			log.Info("[ETCD]: recv reply from service: key: %s, ttl: %d, id: %d", s.Key, resp.TTL, resp.ID)
		}
	}
}

//keep alive
func (s *EtcdServer) keepAlice() (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	resp, err := s.Client.Grant(s.Ctx, EtcdServerGrantTtl) //创建租约
	if err != nil {
		return nil, err
	}

	_, err = s.Client.Put(s.Ctx, s.Key, s.Value, clientv3.WithLease(resp.ID))
	if err != nil {
		return nil, err
	}

	s.LeasdId = resp.ID

	return s.Client.KeepAlive(s.Ctx, resp.ID) //保持租约
}

//释放租约
func (s *EtcdServer) revoke() error {
	_, err := s.Client.Revoke(s.Ctx, s.LeasdId)
	if err != nil {
		return err
	}

	return nil
}

//stop
func (s *EtcdServer) Stop() {
	s.StopState <- errors.New("stop")
}
