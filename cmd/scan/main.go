// escanear2.go
package main

import (
	"encoding/json"
	"escaner/internal/backend"
	"escaner/internal/models"
	scan "escaner/internal/utils"
	"escaner/internal/wsclient"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"time"
)

var (
	timeoutMs   = flag.Int("timeout", 1000, "Timeout en ms para ping / tcp connect")
	portsArg    = flag.String("ports", "22,80,443,3389,445,139,9100,631,515,3306,53,8080,137,161", "Puertos separados por comas para fallback y fingerprint")
	concurrency = flag.Int("c", 200, "Concurrencia máxima para escaneo")
	jsonOut     = flag.Bool("json", false, "Salida JSON en vez de texto")

	// Config backend
	//ipServer = flag.String("ipserver", "192.168.0.24", "direcion del servidor del backend")
	ipServer = flag.String("ipserver", "192.168.182.136", "direcion del servidor del backend")

	//backendURL = flag.String("backend", "http://192.168.182.136:3000/dispositivos/found", "URL del backend para enviar dispositivos")
	//backendURL        = flag.String("backend", "http://192.168.0.24:3000/dispositivos/found", "URL del backend para enviar dispositivos")
	backendTimeoutSec = flag.Int("backend-timeout", 3, "Timeout en segundos para cada POST al backend")
	backendWorkers    = flag.Int("backend-workers", 20, "Concurrencia para envíos al backend")
)

func main() {

	flag.Parse()
	backendURL := fmt.Sprintf("http://%s:3000/dispositivos/found", *ipServer)
	wsURL := fmt.Sprintf("%s:8082", *ipServer)
	ip := fmt.Sprint("", *ipServer)
	//NUEVA FUNCIONALIDAD------------------------------
	backendURLEquipos := fmt.Sprintf("http://%s:3000/equipos", *ipServer)
	equipo := scan.ObtenerInfoEquipo(*ipServer, os.Getenv("USERNAME"))
	go func() {
		for {
			err := backend.EnviarEquipo(equipo, backendURLEquipos, time.Duration(*backendTimeoutSec)*time.Second)
			if err != nil {
				fmt.Println("Error enviando equipo:", err)
			} else {
				fmt.Println("✅ Datos del equipo enviados correctamente al backend")
			}
			time.Sleep(24 * time.Hour) // o cada cierto tiempo que definas
		}
	}()

	//----------------------------

	// Si no hay argumento, arrancamos solo el servidor HTTP (modo agente)
	if flag.NArg() < 1 {
		// go httpserver.RunHTTPServer(
		// 	*portsArg,
		// 	*timeoutMs,
		// 	*concurrency,
		// 	*backendWorkers,
		// 	*backendTimeoutSec,
		// 	backendURL,
		// )

		// 🚀 Iniciar conexión WebSocket
		//go wsclient.ConnectWebSocket("192.168.0.24:8082") // o la IP donde corre tu backend
		//go wsclient.ConnectWebSocket("192.168.182.136:8082") // o la IP donde corre tu backend
		go wsclient.ConnectWebSocket(wsURL, ip)

		fmt.Println("Servidor del agente escuchando en :8081 (modo servidor + WS).")
		//select {}
		// Esperar Ctrl+C
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt

		fmt.Println("🔌 Señal recibida, cerrando proceso...")

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
			err := backend.SendToBackend(r, time.Duration(*backendTimeoutSec)*time.Second, backendURL)
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
