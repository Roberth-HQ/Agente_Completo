package wsclient

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// Estructura base de mensaje que el WS enviará o recibirá
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Función para iniciar la conexión con el servidor WebSocket
func ConnectWebSocket(serverAddr string, ip string) {
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: "/agents"}
	log.Printf("Conectando al servidor WebSocket: %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("❌ Error conectando al backend WebSocket: %v", err)
	}
	defer c.Close()

	done := make(chan struct{})
	endpoint := fmt.Sprintf("http://%s:3000/dispositivos/found", ip)

	// 🧠 Construir el mensaje inicial con datos del sistema
	registerMsg := WSMessage{
		Type: "register",
		Data: map[string]interface{}{
			"agentId":    GetMacAddress(),
			"subnet":     GetLocalSubnet(),
			"cpuCores":   GetCPUCores(),
			"ramMb":      GetRAM(),
			"isFallback": true,
		},
	}
	// 📨 Enviar datos al backend
	c.WriteJSON(registerMsg)
	log.Printf("📤 Agente registrado: %+v\n", registerMsg.Data)

	// Manejar mensajes entrantes del servidor
	go func() {
		defer close(done)

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("Error leyendo mensaje:", err)
				return
			}
			fmt.Printf("📩 Mensaje recibido del servidor: %s\n", message)

			// Aquí puedes interpretar comandos que el backend envía
			// Por ejemplo: {"type": "scan", "data": {"subred": "182"}}
			var msg WSMessage
			if err := json.Unmarshal(message, &msg); err == nil {
				if msg.Type == "scan_request" {
					fmt.Println("🚀 Iniciando escaneo solicitado por WS con data:", msg.Data)
					RunScanFromWS(msg.Data, endpoint, 3)
					//RunScanFromWS(msg.Data, "http://ip:3000/dispositivos/found", 3)
					//RunScanFromWS(msg.Data, "http://192.168.0.24:3000/dispositivos/found", 3)
				}

			}
		}
	}()

	// Esperar señales del sistema para cerrar la conexión
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Enviar un mensaje inicial
	// msg := WSMessage{
	// 	Type: "register",
	// 	Data: map[string]string{"agent_id": "agente-001"},
	// }
	// c.WriteJSON(msg)

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("🔌 Cierre solicitado, desconectando WS...")

			// 🔒 Envía el mensaje de cierre al servidor
			err := c.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("⚠️ Error enviando mensaje de cierre:", err)
			}

			// 🧹 Espera un momento para permitir que el cierre llegue correctamente
			time.Sleep(500 * time.Millisecond)

			// 🔚 Cierra la conexión
			c.Close()
			return
		}
	}
}
