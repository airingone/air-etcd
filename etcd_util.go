package air_etcd

import (
	"errors"
	"fmt"
	"net"
)

//普通功能函数

//将地址信息转化为grpc地址
//info: 地址信息
func ChangeAddrToGrpc(info *ServerInfoSt) string {
	//addr := "http://"
	addr := info.Ip
	addr += ":"
	addr += fmt.Sprint(info.Port)

	return addr
}

//获取本地ip
func GetLoaclIp() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		//fmt.Println(err)
		return "", err
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", errors.New("not exist")
}
