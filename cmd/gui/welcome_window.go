package main

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func showWelcomeWindow() {
	// Intentamos poner el icono
	if iconData := getIcon(); len(iconData) > 0 {
		mainWindow.SetIcon(fyne.NewStaticResource("icono.ico", iconData))
	}

	// Título grande y descriptivo
	title := widget.NewLabelWithStyle(
		"Agente de Inventario - Proyecto LUCY\nGestión y Monitoreo de la Red",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// Botones Next y Cancelar al fondo y al centro
	nextBtn := widget.NewButton("Next", func() {
		showMainContent() // Reemplaza el contenido con la ventana principal
	})
	cancelBtn := widget.NewButton("Cancelar", func() {
		os.Exit(0)
	})
	buttons := container.NewHBox(nextBtn, cancelBtn)
	buttonsCenter := container.NewCenter(buttons)

	// Layout: título arriba, espacio medio y botones abajo
	content := container.NewBorder(
		title,               // arriba
		buttonsCenter,       // abajo
		nil,                 // izquierda
		nil,                 // derecha
		widget.NewLabel(""), // espacio vacío en el centro
	)

	mainWindow.SetContent(content)
	mainWindow.Resize(fyne.NewSize(450, 200))
	mainWindow.Show()
	mainWindow.RequestFocus()
}
