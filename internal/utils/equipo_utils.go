package scan

import (
	"escaner/internal/models"
	"fmt"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/yusufpapurcu/wmi"
)

// --- Estructuras WMI ---set CGO_ENABLED=0
type Win32_ComputerSystemProduct struct {
	UUID string
}

type Win32_ComputerSystem struct {
	Manufacturer string
	Model        string
}

type Win32_BaseBoard struct {
	Manufacturer string
	Product      string
	SerialNumber string
}

type Win32_BIOS struct {
	Manufacturer      string
	SMBIOSBIOSVersion string
	ReleaseDate       string
}

type Win32_VideoController struct {
	Name string
}

type Win32_NetworkAdapterConfiguration struct {
	MACAddress string
	IPEnabled  bool
}

// ObtenerInfoEquipo devuelve un struct Equipo con toda la info
func ObtenerInfoEquipo(ip string, usuario string) models.Equipo {
	// --- gopsutil ---
	h, _ := host.Info()
	cpus, _ := cpu.Info()
	v, _ := mem.VirtualMemory()
	parts, _ := disk.Partitions(false)

	var discos []models.Disk
	for _, part := range parts {
		usage, _ := disk.Usage(part.Mountpoint)
		discos = append(discos, models.Disk{
			Model: part.Device,
			Size:  usage.Total,
			Type:  part.Fstype,
		})
	}

	// --- WMI ---
	var systemProduct []Win32_ComputerSystemProduct
	_ = wmi.Query("SELECT UUID FROM Win32_ComputerSystemProduct", &systemProduct)

	var computer []Win32_ComputerSystem
	err := wmi.Query("SELECT Manufacturer, Model FROM Win32_ComputerSystem", &computer)
	if err != nil {
		fmt.Println("Error WMI ComputerSystem:", err)
	}

	var bios []Win32_BIOS
	_ = wmi.Query("SELECT Manufacturer, SMBIOSBIOSVersion, ReleaseDate FROM Win32_BIOS", &bios)

	var board []Win32_BaseBoard
	_ = wmi.Query("SELECT Manufacturer, Product, SerialNumber FROM Win32_BaseBoard", &board)

	var gpu []Win32_VideoController
	_ = wmi.Query("SELECT Name FROM Win32_VideoController", &gpu)

	var macs []Win32_NetworkAdapterConfiguration
	_ = wmi.Query("SELECT MACAddress, IPEnabled FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled=true", &macs)

	// --- Map de códigos internos a modelo comercial ---
	modelMap := map[string]string{
		"0Y7WYT": "OptiPlex 7040",
		"123ABC": "Vostro 3470",
		// agrega más códigos según tus equipos
	}

	rawModel := safeGet(computer, 0, func(c Win32_ComputerSystem) string { return c.Model })
	commercialModel, ok := modelMap[rawModel]
	if !ok {
		commercialModel = rawModel
	}

	// --- Llenar struct Equipo ---
	eq := models.Equipo{
		UUID:      safeGet(systemProduct, 0, func(p Win32_ComputerSystemProduct) string { return p.UUID }),
		MAC:       safeGet(macs, 0, func(n Win32_NetworkAdapterConfiguration) string { return n.MACAddress }),
		Hostname:  h.Hostname,
		OS:        h.Platform,
		OSVersion: h.PlatformVersion,
		CPU:       safeGet(cpus, 0, func(c cpu.InfoStat) string { return c.ModelName }),
		RAM:       fmt.Sprintf("%.2f GB", float64(v.Total)/1024/1024/1024),
		Disk:      discos,
		BIOS: models.BIOS{
			Manufacturer: safeGet(bios, 0, func(b Win32_BIOS) string { return b.Manufacturer }),
			Version:      safeGet(bios, 0, func(b Win32_BIOS) string { return b.SMBIOSBIOSVersion }),
			Date:         safeGet(bios, 0, func(b Win32_BIOS) string { return b.ReleaseDate }),
		},
		Motherboard: models.Motherboard{
			Manufacturer: safeGet(computer, 0, func(c Win32_ComputerSystem) string { return c.Manufacturer }),
			Product:      commercialModel,
			SerialNumber: safeGet(board, 0, func(b Win32_BaseBoard) string { return b.SerialNumber }),
		},
		GPU:       safeGet(gpu, 0, func(g Win32_VideoController) string { return g.Name }),
		IP:        ip,
		Usuario:   usuario,
		CodActivo: fmt.Sprintf("%s %s", safeGet(computer, 0, func(c Win32_ComputerSystem) string { return c.Manufacturer }), commercialModel),
	}

	return eq
}

// helper para evitar panic en slices vacíos
func safeGet[T any, R any](slice []T, index int, f func(T) R) R {
	var zero R
	if len(slice) > index {
		return f(slice[index])
	}
	return zero
}
