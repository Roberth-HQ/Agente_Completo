package backend

import (
	"bytes"
	"encoding/json"
	"escaner/internal/models"
	scan "escaner/internal/utils"
	"fmt"
	"io"
	"net/http"
	"time"
)

// func SendToBackendBatch(results []models.Result, workers int, timeout time.Duration, backendURL string) {
// 	client := &http.Client{Timeout: timeout}
// 	sem := make(chan struct{}, workers)
// 	var wg sync.WaitGroup

// 	for _, r := range results {
// 		dto := map[string]string{
// 			"ip":     r.IP,
// 			"alive":  "sí",
// 			"via":    r.Method,
// 			"device": ifEmpty(r.DeviceType, "Unknown"),
// 			"mac":    r.MAC,
// 			"name":   r.ReverseDNS,
// 		}

// 		body, err := json.Marshal(dto)
// 		if err != nil {
// 			fmt.Printf("Error marshal DTO %s: %v\n", r.IP, err)
// 			continue
// 		}

// 		wg.Add(1)
// 		sem <- struct{}{}
// 		go func(ip string, payload []byte) {
// 			defer wg.Done()
// 			defer func() { <-sem }()

// 			req, err := http.NewRequest("POST", backendURL, bytes.NewBuffer(payload))
// 			if err != nil {
// 				fmt.Printf("Error creando request para %s: %v\n", ip, err)
// 				return
// 			}
// 			req.Header.Set("Content-Type", "application/json")

// 			resp, err := client.Do(req)
// 			if err != nil {
// 				fmt.Printf("Error POST backend %s: %v\n", ip, err)
// 				return
// 			}
// 			defer resp.Body.Close()

// 			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
// 				b, _ := io.ReadAll(resp.Body)
// 				fmt.Printf("Backend respondió error para %s: %d - %s\n", ip, resp.StatusCode, string(b))
// 				return
// 			}
// 			fmt.Printf("Enviado (OK): %s\n", ip)
// 		}(r.IP, body)
// 	}

// 	wg.Wait()
// }

func ifEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// ///individuañ////
func SendToBackend(r models.Result, timeout time.Duration, backendURL string) error {
	client := &http.Client{Timeout: timeout}

	fmt.Println("Dispositivooooooooooooooo vivoooooooo detectadoooooooooooooooooooo:", scan.FormatResult(r))

	dto := map[string]string{
		"ip":     r.IP,
		"alive":  "sí",
		"via":    r.Method,
		"device": ifEmpty(r.DeviceType, "Unknown"),
		"mac":    r.MAC,
		"name":   r.ReverseDNS,
	}

	body, err := json.Marshal(dto)
	if err != nil {
		return fmt.Errorf("error marshal DTO %s: %v", r.IP, err)
	}

	req, err := http.NewRequest("POST", backendURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error creando request para %s: %v", r.IP, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error POST backend %s: %v", r.IP, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("backend respondió error para %s: %d - %s", r.IP, resp.StatusCode, string(b))
	}

	fmt.Printf("Enviado (OK): %s\n", r.IP)
	return nil
}
