package air_etcd

import (
	"context"
	"encoding/json"
	"github.com/airingone/log"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc/resolver"
)

//为grpc封装的解析器
type GrpcResolver struct {
	Endpoints   []string
	ServiceName string
	Client      *clientv3.Client
	Cc          resolver.ClientConn
	AddrDict    map[string]resolver.Address
}

//Create
func NewGrpcResolver( /*serviceName string,*/ endpoints []string) *GrpcResolver {
	r := &GrpcResolver{
		//ServiceName: serviceName,
		Endpoints: endpoints,
		AddrDict:  make(map[string]resolver.Address),
	}
	return r
}

//Scheme
func (r *GrpcResolver) Scheme() string {
	return r.ServiceName
}

//build
func (r *GrpcResolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	cli, err := NewEtcd(r.Endpoints)
	if err != nil {
		return nil, err
	}
	r.ServiceName = target.Endpoint //	即为grpc Dail函数的target参数
	log.Info("Build: target: %+v", target)
	r.Cc = cc
	r.Client = cli

	go r.watch()

	return r, nil
}

// ResolveNow
func (r *GrpcResolver) ResolveNow(rn resolver.ResolveNowOptions) {
}

//Close
func (r *GrpcResolver) Close() {
}

//watch
func (r *GrpcResolver) watch() {
	defer func() {
		if r := recover(); r != nil {
			log.PanicTrack()
		}
	}()

	resp, err := r.Client.Get(context.Background(), r.ServiceName, clientv3.WithPrefix())
	if err == nil {
		for i, kv := range resp.Kvs {
			info := &ServerInfoSt{}
			err := json.Unmarshal([]byte(kv.Value), info)
			if err == nil {
				r.AddrDict[string(resp.Kvs[i].Key)] = resolver.Address{Addr: ChangeAddrToGrpc(info)}
			}
			r.update()
		}
	}

	rch := r.Client.Watch(context.Background(), r.ServiceName, clientv3.WithPrefix(), clientv3.WithPrevKV())
	for n := range rch {
		for _, ev := range n.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				info := &ServerInfoSt{}
				err := json.Unmarshal([]byte(ev.Kv.Value), info)
				if err != nil {
				} else {
					r.AddrDict[string(ev.Kv.Key)] = resolver.Address{Addr: ChangeAddrToGrpc(info)}
					r.update()
				}
			case clientv3.EventTypeDelete:
				delete(r.AddrDict, string(ev.PrevKv.Key))
				r.update()
			}
		}
	}
}

//触发grpc更新路由地址
func (r *GrpcResolver) update() {
	var state resolver.State
	for _, v := range r.AddrDict {
		state.Addresses = append(state.Addresses, v)
	}
	r.Cc.UpdateState(state)
	log.Info("update: addr: %+v", state)
}

/*
type GrpcBuilder struct {
}
*/
