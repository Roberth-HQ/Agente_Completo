package wsclient

import (
	"encoding/json"
	"escaner/internal/backend"
	"escaner/internal/models"
	scan "escaner/internal/utils"
	"fmt"
	"strconv"
	"time"
)

// Estructura del mensaje WS esperado
type ScanRequest struct {
	Subred string `json:"subnet"`
}

// Función que ejecuta el escaneo cuando llega por WS
func RunScanFromWS(data interface{}, backendURL string, backendTimeoutSec int) {
	bytes, _ := json.Marshal(data)
	var req ScanRequest
	if err := json.Unmarshal(bytes, &req); err != nil {
		fmt.Println("❌ Error interpretando datos de escaneo WS:", err)
		return
	}

	subredInt, err := strconv.Atoi(req.Subred)
	if err != nil {
		fmt.Println("❌ Subred inválida:", req.Subred)
		return
	}

	ipRange := fmt.Sprintf("192.168.%d.1-255", subredInt)
	fmt.Printf("🚀 Escaneo iniciado desde WS: %s\n", ipRange)

	ips, err := scan.ExpandArgToIPs(ipRange)
	if err != nil {
		fmt.Println("❌ Error generando IPs:", err)
		return
	}

	ports := scan.ParsePorts("22,80,443,3389,445,139,9100,631,515,3306,53,8080,137,161")
	timeout := 1 * time.Second

	onAlive := func(r models.Result) {
		fmt.Println("📡 Dispositivo detectado:", scan.FormatResult(r))
		err := backend.SendToBackend(r, time.Duration(backendTimeoutSec)*time.Second, backendURL)
		if err != nil {
			fmt.Println("❌ Error enviando al backend:", err)
		}
	}

	scan.ScanIPs(ips, ports, timeout, 200, onAlive)
	fmt.Println("✅ Escaneo WS completado.")
}
