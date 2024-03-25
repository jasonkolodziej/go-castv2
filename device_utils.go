package castv2

import (
	"fmt"
	"net"
)

/*
discoverLocalInterfaces

	disovers interfaces used by the device executing this function
*/
func DiscoverLocalInterfaces() []net.Interface {
	var ret []net.Interface
	netFaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, face := range netFaces {
		addrs, err := face.Addrs()
		if err != nil {
			panic(err)
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
				fmt.Println(ipNet.IP)
				ret = append(ret, face)
			}
		}
	}
	return ret
}
