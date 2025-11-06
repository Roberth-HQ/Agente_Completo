package main

import (
	"escaner/internal/wsclient"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ---- Ventana WS ----
// ---- Ventana WS ----
func showWSContent() {
	var defChk, normChk *widget.Check

	defChk = widget.NewCheck("Por defecto", func(b bool) {
		if b {
			isFallback = true
			if normChk != nil {
				normChk.SetChecked(false)
			}
		} else if normChk != nil && !normChk.Checked {
			isFallback = false
		}
	})

	normChk = widget.NewCheck("Normal", func(b bool) {
		if b {
			isFallback = false
			if defChk != nil {
				defChk.SetChecked(false)
			}
		} else if defChk != nil && !defChk.Checked {
			isFallback = true
		}
	})

	atrasBtn := widget.NewButton("Atrás", func() {
		showMainContent()
	})

	siguienteBtn := widget.NewButton("Siguiente", func() {
		// Conectarse al WS antes de cerrar la ventana
		wsURL := fmt.Sprintf("%s:8082", ipServer)
		ip := ipServer // aquí la IP del servidor
		go wsclient.ConnectWebSocket(wsURL, ip, isFallback)

		showExitWindow()
	})

	// Contenedor horizontal centrado para los botones
	buttons := container.NewHBox(atrasBtn, siguienteBtn)
	buttonsCentered := container.NewCenter(buttons)

	// Contenido principal
	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("IP del servidor: %s", ipServer)),
		defChk,
		normChk,
	)

	mainWindow.SetContent(container.NewBorder(nil, buttonsCentered, nil, nil, content))
	mainWindow.Resize(fyne.NewSize(500, 300))
	mainWindow.Show()
	mainWindow.RequestFocus()
}
