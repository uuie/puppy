package config

import (
	"net"
	"path/filepath"
)

func getLocalIp() string {
	ifaces, _ := net.Interfaces()
	// handle err
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		var ip net.IP
		// handle err
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP

			}

		}
		if m, _ := filepath.Match(Instance.InboundIpPattern, ip.String()); m {
			return ip.String()
		}
	}
	return ""
}
