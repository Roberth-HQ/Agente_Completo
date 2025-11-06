package main

import (
	"fmt"
	"os"
	"time"

	"escaner/internal/backend"
	scan "escaner/internal/utils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/getlantern/systray"
)

var (
	a          fyne.App
	mainWindow fyne.Window
)

func main() {
	// Creamos la app de Fyne en el hilo principal
	a = app.New()

	// Iniciamos el Systray en segundo plano
	go startTray()

	// Mostramos la ventana de bienvenida
	showWelcomeWindow()

	// Ejecutamos Fyne
	a.Run()
}

// ---- Systray ----
func startTray() {
	onReady := func() {
		systray.SetIcon(getIcon())
		systray.SetTitle("Agente GUI")
		systray.SetTooltip("Agente de Inventario en segundo plano")

		mOpen := systray.AddMenuItem("Abrir", "Abrir la ventana principal")
		mQuit := systray.AddMenuItem("Salir", "Cerrar la aplicación")

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					openMainWindow()
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

func getIcon() []byte {
	data, err := os.ReadFile("icono.ico")
	if err != nil {
		fmt.Println("Error cargando icono:", err)
		return []byte{}
	}
	return data
}

// ---- Ventana de bienvenida ----
func showWelcomeWindow() {
	w := a.NewWindow("Bienvenido - Agente GUI")

	label := widget.NewLabel("Bienvenido al Agente de Inventario")
	nextBtn := widget.NewButton("Next", func() {
		w.Close()
		openMainWindow()
	})
	cancelBtn := widget.NewButton("Cancelar", func() {
		w.Close()
		systray.Quit()
		os.Exit(0)
	})

	w.SetContent(container.NewVBox(label, nextBtn, cancelBtn))
	w.Resize(fyne.NewSize(300, 150))
	w.Show()
}

// ---- Ventana principal ----
func openMainWindow() {
	if mainWindow != nil {
		mainWindow.Show()
		mainWindow.RequestFocus()
		return
	}

	mainWindow = a.NewWindow("Agente GUI")

	ipEntry := widget.NewEntry()
	ipEntry.SetPlaceHolder("Ingrese la IP del servidor")
	statusLabel := widget.NewLabel("Esperando acción...")

	sendButton := widget.NewButton("Enviar Info del Equipo", func() {
		ip := ipEntry.Text
		if ip == "" {
			statusLabel.SetText("❌ Por favor ingrese la IP del servidor")
			return
		}

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

	mainWindow.SetContent(container.NewVBox(
		widget.NewLabel("Agente de Inventario - GUI"),
		ipEntry,
		sendButton,
		statusLabel,
	))

	mainWindow.Resize(fyne.NewSize(400, 200))
	mainWindow.Show()
}
