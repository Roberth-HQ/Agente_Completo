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

func SendFinalMessage(timeout time.Duration, backendURL string, subred string) error {
	client := &http.Client{Timeout: timeout}

	finalDto := map[string]string{
		"status":  "ok",
		"message": "finalizado",
		"subred":  subred, // ✅ enviar la subred
	}

	body, err := json.Marshal(finalDto)
	if err != nil {
		return fmt.Errorf("error marshal mensaje final: %v", err)
	}

	req, err := http.NewRequest("POST", backendURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error creando request final: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error enviando mensaje final: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backend respondió con código: %d", resp.StatusCode)
	}

	fmt.Println("✅ Mensaje final enviado correctamente al backend")
	return nil
}
