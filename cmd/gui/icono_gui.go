package main

import (
	"fmt"
	"os"

	"github.com/getlantern/systray"
)

// getIcon devuelve los bytes del icono de la aplicación
func getIcon() []byte {
	data, err := os.ReadFile("icono.ico") // Asegúrate de tener icono.ico en la misma carpeta
	if err != nil {
		fmt.Println("Error cargando icono:", err)
		return []byte{}
	}
	return data
}

// ---- Systray ----
func startTray() {
	onReady := func() {
		systray.SetIcon(getIcon()) // Establece el icono
		systray.SetTitle("Agente GUI")
		systray.SetTooltip("Agente de Inventario en segundo plano")

		// Menú del tray
		//mOpen := systray.AddMenuItem("Abrir", "Abrir la ventana principal")
		mQuit := systray.AddMenuItem("Salir", "Cerrar la aplicación")

		// Manejo de clicks
		go func() {
			for {
				select {
				//case <-mOpen.ClickedCh:
				//openMainWindow() // Abre la ventana principal
				case <-mQuit.ClickedCh:
					systray.Quit()
					os.Exit(0)
				}
			}
		}()
	}

	onExit := func() {
		fmt.Println("Systray cerrado")
	}

	systray.Run(onReady, onExit)
}
