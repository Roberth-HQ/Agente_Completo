go get github.com/gorilla/websocket
go get github.com/joho/godotenv
go get github.com/shirou/gopsutil/v3
go mod tidy
con registro de equipos:
go get github.com/shirou/gopsutil/v3@latest
go mod tidy
interface grafica:
go mod init github.com/tuusuario/tuagente
go get fyne.io/fyne/v2
go mod tidy
go get fyne.io/fyne/v2/internal/svg@v2.7.0
go get fyne.io/fyne/v2/internal/painter@v2.7.0
go get fyne.io/fyne/v2/lang@v2.7.0
dependencias de la intalacion grafica
go env -w CGO_ENABLED=1

gcc --version
go env CGO_ENABLED
    go get github.com/getlantern/systray



go run ./cmd/scan
go build -o agente_gui.exe ./cmd/gui
go build -tags "gl,windows" -o agente_gui.exe ./cmd/gui
