package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

var (
	a          fyne.App
	mainWindow fyne.Window
	wsWindow   fyne.Window

	isFallback = true
	ipServer   = "192.168.182.136" // IP por defecto
)

func main() {
	a = app.New()

	// Inicia systray en segundo plano
	go startTray()

	// Crear ventana principal invisible
	mainWindow = a.NewWindow("Agente Lucy")
	mainWindow.Hide()

	// Mostrar contenido de bienvenida
	showWelcomeWindow()

	// Mantener la app viva
	a.Run()
}
