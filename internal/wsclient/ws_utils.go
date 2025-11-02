package wsclient

import (
	"net"
	"runtime"

	"github.com/shirou/gopsutil/mem"
)

// Obtener la primera MAC disponible del sistema
func GetMacAddress() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}
	for _, i := range ifaces {
		if i.Flags&net.FlagUp != 0 && len(i.HardwareAddr) > 0 {
			return i.HardwareAddr.String()
		}
	}
	return "unknown"
}

// Obtener la subred local (ejemplo simplificado)
func GetLocalSubnet() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String() + "/24"
			}
		}
	}
	return "unknown"
}

// Obtener cantidad de núcleos de CPU
func GetCPUCores() int {
	return runtime.NumCPU()
}

// Obtener memoria RAM total en MB
func GetRAM() uint64 {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0
	}
	return v.Total / 1024 / 1024
}
