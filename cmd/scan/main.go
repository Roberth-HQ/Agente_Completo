// escanear2.go
package main

import (
	"encoding/json"
	"escaner/internal/backend"
	httpserver "escaner/internal/http_server"
	"escaner/internal/models"
	scan "escaner/internal/utils"
	"flag"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

var (
	timeoutMs   = flag.Int("timeout", 1000, "Timeout en ms para ping / tcp connect")
	portsArg    = flag.String("ports", "22,80,443,3389,445,139,9100,631,515,3306,53,8080,137,161", "Puertos separados por comas para fallback y fingerprint")
	concurrency = flag.Int("c", 200, "Concurrencia máxima para escaneo")
	jsonOut     = flag.Bool("json", false, "Salida JSON en vez de texto")

	// Config backend
	backendURL        = flag.String("backend", "http://192.168.182.136:3000/dispositivos", "URL del backend para enviar dispositivos")
	backendTimeoutSec = flag.Int("backend-timeout", 3, "Timeout en segundos para cada POST al backend")
	backendWorkers    = flag.Int("backend-workers", 20, "Concurrencia para envíos al backend")
)

func main() {
	flag.Parse()

	// Si no hay argumento, arrancamos solo el servidor HTTP (modo agente)
	if flag.NArg() < 1 {
		go httpserver.RunHTTPServer(
			*portsArg,
			*timeoutMs,
			*concurrency,
			*backendWorkers,
			*backendTimeoutSec,
			*backendURL,
		)
		fmt.Println("Servidor del agente escuchando en :8081 (modo solo servidor).")
		// Mantener proceso vivo
		select {}
	}

	// Si sí hay argumento, ejecutamos flujo CLI: escanear -> imprimir -> enviar
	arg := flag.Arg(0)
	ips, err := scan.ExpandArgToIPs(arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error procesando argumento: %v\n", err)
		os.Exit(1)
	}
	ports := scan.ParsePorts(*portsArg)
	timeout := time.Duration(*timeoutMs) * time.Millisecond

	var aliveCount int64 = 0

	// Callback que se llama en cada resultado escaneado
	onAlive := func(r models.Result) {
		if r.Alive {
			atomic.AddInt64(&aliveCount, 1)
			fmt.Println("funcion del main")
			fmt.Println("Dispositivooooooooooo vivo detectado:", scan.FormatResult(r))
			err := backend.SendToBackend(r, time.Duration(*backendTimeoutSec)*time.Second, *backendURL)
			if err != nil {
				fmt.Println("Error enviando al backend:", err)
			}
		}
	}

	// Escaneo paralelo con callback para manejar resultados en vivo
	results := scan.ScanIPs(ips, ports, timeout, *concurrency, onAlive)

	// Output CLI completo
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
	} else {
		for _, r := range results {
			fmt.Println(scan.FormatResult(r))
		}
	}

	fmt.Printf("Escaneo completado. Dispositivos vivos enviados: %d\n", aliveCount)
}
