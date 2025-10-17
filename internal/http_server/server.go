package httpserver

import (
	"encoding/json"
	"escaner/internal/backend"
	"escaner/internal/models"
	scan "escaner/internal/utils"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// runHTTPServer levanta el servidor HTTP que acepta POST /scan {"subred":"183"}
func RunHTTPServer(
	portsArg string,
	timeoutMs int,
	concurrency int,
	backendWorkers int,
	backendTimeoutSec int,
	backendURL string,
) {
	http.HandleFunc("/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Solo se permite POST", http.StatusMethodNotAllowed)
			return
		}

		type ScanRequest struct {
			Subred string `json:"subred"`
		}

		var req ScanRequest
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&req)
		if err != nil || req.Subred == "" {
			http.Error(w, "Body inválido. Esperado: {\"subred\": \"183\"}", http.StatusBadRequest)
			return
		}

		subredInt, err := strconv.Atoi(req.Subred)
		if err != nil || subredInt < 0 || subredInt > 255 {
			http.Error(w, "Subred inválida", http.StatusBadRequest)
			return
		}

		ipRange := fmt.Sprintf("192.168.%d.1-255", subredInt)
		fmt.Printf("Escaneando subred: %s\n", ipRange)

		ips, err := scan.ExpandArgToIPs(ipRange)
		if err != nil {
			http.Error(w, "Error generando IPs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		ports := scan.ParsePorts(portsArg)
		timeout := time.Duration(timeoutMs) * time.Millisecond

		// Escanear primero
		results := scan.ScanIPs(ips, ports, timeout, concurrency)

		// Imprimir resultados (rápido)
		for _, res := range results {
			fmt.Println(scan.FormatResult(res))
		}

		// Recolectar vivos y enviarlos en lote (esperamos a que termine)
		var aliveResults []models.Result
		for _, r := range results {
			if r.Alive {
				aliveResults = append(aliveResults, r)
			}
		}
		if len(aliveResults) > 0 {
			fmt.Printf("Enviando %d dispositivos vivos al backend...\n", len(aliveResults))
			backend.SendToBackendBatch(aliveResults, backendWorkers, time.Duration(backendTimeoutSec)*time.Second, backendURL)
		} else {
			fmt.Println("No hay dispositivos vivos para enviar.")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Escaneo completado. Dispositivos vivos: %d", len(aliveResults))))
	})

	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Fprintf(os.Stderr, "error servidor HTTP: %v\n", err)
	}
}
