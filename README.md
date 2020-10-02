# etcd client组件
## 1.组件描述
etcd client客户端，分别实现common client etcd client，common server etcd client，grpc client etcd client。 

## 2.如何使用
### 2.1 服务注册
```
import (
    "github.com/airingone/config"
    air_etcd "github.com/airingone/air-etcd"
)

func main() {
    config.InitConfig()                        //进程启动时调用一次初始化配置文件，配置文件名为config.yml，目录路径为../conf/或./
    log.InitLog(config.GetLogConfig("log"))    //进程启动时调用一次初始化日志
    
    //进程初始化启动一次etcd client
    air_etcd.RegisterLocalServerToEtcd(config.GetServerConfig("server").name, 
        8080, 
        config.GetEtcdConfig("etcd").Addrs)
    select {} 
}
```
### 2.2 服务发现
```
import (
    "github.com/airingone/config"
    air_etcd "github.com/airingone/air-etcd"
)

func main() {
    config.InitConfig()                        //进程启动时调用一次初始化配置文件，配置文件名为config.yml，目录路径为../conf/或./
    log.InitLog(config.GetLogConfig("log"))    //进程启动时调用一次初始化日志
    
    //进程初始化启动一次etcd client
    air_etcd.InitEtcdClient(config.GetEtcdConfig("etcd"),
        config.GetNetConfig("udp_client").Addr)
	defer air_etcd.CloseEtcdClient()
    time.Sleep(1 * time.Second)

    //获取client
    client, err := air_etcd.GetEtcdClientByServerName(config.GetNetConfig("udp_client").ServerName)
    addrs, err := client.GetAllServerAddr() //获取全部服务地址
    if err == nil {
        log.Error("[ETCD]: addrs %+v", addrs)
    }

    addrs, err:= client.RandGetServerAddr() //随机获取一个服务地址
    if err == nil {
        log.Error("[ETCD]: addrs %+v", addrs)
    }
    select {} 
}
```

### 2.3 grpc服务发现
```
import (
    "github.com/airingone/config"
    "google.golang.org/grpc"
    "google.golang.org/grpc/balancer/roundrobin"
    "google.golang.org/grpc/resolver"
    air_etcd "github.com/airingone/air-etcd"
)

func main() {
    config.InitConfig()                        //进程启动时调用一次初始化配置文件，配置文件名为config.yml，目录路径为../conf/或./
    log.InitLog(config.GetLogConfig("log"))    //进程启动时调用一次初始化日志
    
    //每个服务全局注册一次
    etcdConfig := config.GetGrpcConfig("grpc_test")
    r := airetcd.NewGrpcResolver(config.GetEtcdConfig("etcd").Addrs)
        resolver.Register(r)
    
    //conn初始化一次即可，grpc会维护连接
    ctx, _ := context.WithTimeout(context.Background(), time.Duration(etcdConfig.TimeOutMs)*time.Millisecond)
    conn, err := grpc.DialContext(ctx, etcdConfig.Name, //obejct会传给etcd作为watch对象
    	grpc.WithInsecure(),
    	grpc.WithDefaultServiceConfig(roundrobin.Name),
    	grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
    	grpc.WithBlock())
    defer conn.Close()

    //业务逻辑发起服务请求
    //参考https://github.com/airingone/air-grpc/blob/master/grpc_test.go
}
```