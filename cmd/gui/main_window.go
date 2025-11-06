package main

import (
	"escaner/internal/backend"
	scan "escaner/internal/utils"
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func showMainContent() {
	ipEntry := widget.NewEntry()
	ipEntry.SetText(ipServer)
	ipEntry.SetPlaceHolder("Ingrese la IP del servidor")

	statusLabel := widget.NewLabel("Esperando acción...")

	sendButton := widget.NewButton("Enviar Info del Equipo", func() {
		ip := ipEntry.Text
		if ip == "" {
			statusLabel.SetText("❌ Por favor ingrese la IP del servidor")
			return
		}

		ipServer = ip
		statusLabel.SetText("Enviando info...")
		equipo := scan.ObtenerInfoEquipo(ip, os.Getenv("USERNAME"))
		backendURL := fmt.Sprintf("http://%s:3000/equipos", ip)

		go func() {
			err := backend.EnviarEquipo(equipo, backendURL, 5*time.Second)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("❌ Error: %v", err))
			} else {
				statusLabel.SetText("✅ Datos enviados correctamente")
			}
		}()
	})

	// Botones inferiores
	atrasBtn := widget.NewButton("Atrás", func() {
		showWelcomeWindow()
	})

	siguienteBtn := widget.NewButton("Siguiente", func() {
		showExitWindow()
	})

	wsBtn := widget.NewButton("Agente de Monitoreo", func() {
		showWSContent()
	})

	bottomButtons := container.NewHBox(atrasBtn, siguienteBtn, wsBtn)
	bottomButtonsCentered := container.NewCenter(bottomButtons)

	// Contenedor principal
	content := container.NewBorder(nil, bottomButtonsCentered, nil, nil,
		container.NewVBox(
			widget.NewLabelWithStyle(
				"Agente de Inventario del Proyecto Lucy\nGestión de Inventario y Monitoreo de la Red",
				fyne.TextAlignCenter,
				fyne.TextStyle{Bold: true},
			),
			ipEntry,
			sendButton,
			statusLabel,
		),
	)

	mainWindow.SetContent(content)
	mainWindow.Resize(fyne.NewSize(450, 300))
	mainWindow.Show()
	mainWindow.RequestFocus()
}

// Ventana de despedida, también sobre mainWindow
func showExitWindow() {
	label := widget.NewLabel("¡Gracias por usar el Agente de Inventario!\nHasta pronto.")
	finalizarBtn := widget.NewButton("Finalizar", func() {
		os.Exit(0)
	})

	content := container.NewVBox(label, finalizarBtn)
	mainWindow.SetContent(content)
	mainWindow.Resize(fyne.NewSize(350, 150))
	mainWindow.Show()
	mainWindow.RequestFocus()
}
