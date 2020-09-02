package air_etcd

import "fmt"

func ChangeAddrToGrpc(info *ServerInfoSt) string {
	addr := "http://"
	addr += info.Ip
	addr += ":"
	addr += fmt.Sprint(info.Port)

	return addr
}
