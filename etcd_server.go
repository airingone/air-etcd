package air_etcd

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/airingone/log"
	"go.etcd.io/etcd/clientv3"
)

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
	Ctx        context.Context
	Client     *clientv3.Client
	ServerInfo ServerInfoSt
	Key        string
	Value      string
	LeasdId    clientv3.LeaseID
	StopState  chan error
}

//注册服务到etcd
func RegisterLocalServerToEtcd(serverName string, port uint32, etcdEndpoints []string) {
	var info ServerInfoSt
	info.Name = serverName
	info.Port = port
	ip, err := GetLoaclIp()
	if err != nil {
		log.Fatal("RegisterLocalServerToEtcd: GetLoaclIp err")
		return
	}
	info.Ip = ip

	etcdClient, err := NewEtcdServer(info, etcdEndpoints)
	if err != nil {
		log.Fatal("RegisterLocalServerToEtcd: GetLoaclIp err")
		return
	}

	etcdClient.Start()
}

//创建一个etcd client，endpoints为etcd的地址列表
func NewEtcdServer(serverInfo ServerInfoSt, endpoints []string) (*EtcdServer, error) {
	cli, err := NewEtcd(endpoints)
	if err != nil {
		return nil, err
	}

	key := serverInfo.Name + "/" + serverInfo.Ip
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

func (s *EtcdServer) startProcess() error {
	defer func() {
		if r := recover(); r != nil {
			log.PanicTrack()
		}
	}()

	defer s.Client.Close()
	ch, err := s.keepAlice()
	if err != nil {
		return err
	}

	for {
		select {
		case err := <-s.StopState:
			log.Info("ETCD: Stop")
			return err
		case <-s.Client.Ctx().Done():
			log.Info("ETCD: Ctx Done")
			return errors.New("server closed")
		case resp, ok := <-ch:
			if !ok { //租约已关闭
				log.Info("ETCD: Revoke, err:%+v", ok)
				return s.revoke()
			}
			log.Info("ETCD: recv reply from service: key: %s, ttl: %d, id: %d", s.Key, resp.TTL, resp.ID)
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
