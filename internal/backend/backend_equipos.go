package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"escaner/internal/models"
)

// EnviarEquipo envía los datos completos del equipo al backend
func EnviarEquipo(equipo models.Equipo, backendURL string, timeout time.Duration) error {
	jsonData, err := json.Marshal(equipo)
	if err != nil {
		return fmt.Errorf("error al serializar equipo: %w", err)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Post(backendURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error al hacer POST al backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("backend respondió con status %d", resp.StatusCode)
	}

	return nil
}
