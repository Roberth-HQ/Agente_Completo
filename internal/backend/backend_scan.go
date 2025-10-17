package backend

import (
	"encoding/json"
	"escaner/internal/models"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ----------------------- envío en lote al backend -------------------------

// sendToBackendBatch envía los resultados vivos al backend en paralelo controlado.
// Espera a que todos los envíos terminen antes de retornar.
func SendToBackendBatch(results []models.Result, workers int, timeout time.Duration, backendURL string) {
	client := &http.Client{Timeout: timeout}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, r := range results {
		// preparar DTO sencillo (puedes expandir campos según backend)
		dto := map[string]string{
			"ip":    r.IP,
			"alive": "sí",
			"via":   r.Method,
			"divice": func() string {
				if r.DeviceType == "" {
					return "Unknown"
				}
				return r.DeviceType
			}(),
			"mac":  r.MAC,
			"name": r.ReverseDNS,
		}

		body, err := json.Marshal(dto)
		if err != nil {
			fmt.Printf("Error marshal DTO %s: %v\n", r.IP, err)
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(ip string, payload []byte) {
			defer wg.Done()
			defer func() { <-sem }()

			req, err := http.NewRequest("POST", backendURL, strings.NewReader(string(payload)))
			if err != nil {
				fmt.Printf("Error creando request para %s: %v\n", ip, err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error POST backend %s: %v\n", ip, err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(resp.Body)
				fmt.Printf("Backend respondió error para %s: %d - %s\n", ip, resp.StatusCode, string(b))
				return
			}
			fmt.Printf("Enviado (OK): %s\n", ip)
		}(r.IP, body)
	}

	wg.Wait()
}
