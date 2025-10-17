package scan

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ----------------------- MAC retrieval (heurístico) -------------------------

// getMAC intenta obtener la MAC para una IP. Primero fuerza una acción para rellenar la caché ARP,
// luego lee /proc/net/arp (Linux) o ejecuta `arp` y parsea la salida.
func getMAC(ip string, timeout time.Duration) (string, error) {
	// Intentar forzar entrada ARP (ping o TCP) para que la tabla ARP tenga algo
	ensureARPEntry(ip, timeout)

	// Preferir lectura nativa en Linux (/proc/net/arp)
	if runtime.GOOS == "linux" {
		if m := readMACFromProcNetARP(ip); m != "" {
			return m, nil
		}
	}
	// Fallback a comando arp
	if m := macFromARPCommand(ip); m != "" {
		return strings.ToLower(m), nil
	}
	return "", errors.New("MAC no encontrada")
}

// ensureARPEntry intenta provocar que el sistema cree/actualice la entrada ARP de la IP.
// Se usa un ping y/o un intento TCP breve.
func ensureARPEntry(ip string, timeout time.Duration) {
	// Intentamos ICMP primero (rápido)
	if tryPing(ip, timeout) {
		return
	}
	// Intentamos varios puertos TCP comunes para provocar ARP
	common := []int{80, 22, 443}
	for _, p := range common {
		if tryTCP(ip, p, timeout/2) {
			return
		}
	}
	// En Linux, también podemos intentar `arping` si está instalado (no obligatorio)
	if runtime.GOOS == "linux" {
		_ = exec.Command("arping", "-c", "1", "-w", fmt.Sprintf("%d", timeout.Milliseconds()/1000), ip).Run()
	}
}

// readMACFromProcNetARP lee /proc/net/arp en Linux y extrae la MAC si existe
func readMACFromProcNetARP(ip string) string {
	f, err := os.Open("/proc/net/arp")
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	// la primera línea suele ser header; la saltamos si contiene "IP address"
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "IP address") || strings.Contains(line, "Address") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[0] == ip {
			mac := fields[3]
			if mac == "00:00:00:00:00:00" || mac == "00-00-00-00-00-00" {
				return ""
			}
			return strings.ToLower(strings.ReplaceAll(mac, "-", ":"))
		}
	}
	return ""
}

func macFromARPCommand(ip string) string {
	var out []byte
	var err error
	if runtime.GOOS == "windows" {
		out, err = exec.Command("arp", "-a").CombinedOutput()
	} else {
		// en *nix intentamos `arp -n <ip>` que a veces da menos líneas
		out, err = exec.Command("arp", "-n", ip).CombinedOutput()
		if err != nil || len(out) == 0 {
			// fallback a `arp -n`
			out, err = exec.Command("arp", "-n").CombinedOutput()
		}
	}
	if err != nil {
		return ""
	}
	text := string(out)

	// Primero intentamos parsear líneas que contengan la IP
	if mac, err := parseArpOutput(text, ip); err == nil {
		return mac
	}

	// Si no, intentamos buscar cualquier token que parezca MAC en toda la salida
	fields := strings.Fields(text)
	for i, f := range fields {
		clean := strings.Trim(f, "[],()")
		if looksLikeMAC(clean) {
			return strings.ToLower(strings.ReplaceAll(clean, "-", ":"))
		}
		// si encontramos el ip token, miramos el siguiente
		if strings.Contains(clean, ip) && i+1 < len(fields) {
			cand := strings.Trim(fields[i+1], "[],()")
			if looksLikeMAC(cand) {
				return strings.ToLower(strings.ReplaceAll(cand, "-", ":"))
			}
		}
	}
	return ""
}
func looksLikeMAC(s string) bool {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ':' || r == '-'
	})
	if len(parts) != 6 {
		return false
	}
	for _, p := range parts {
		if len(p) != 2 {
			return false
		}
		for _, ch := range p {
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
	}
	return true
}

// parseArpOutput intenta extraer MAC de la salida de `arp` de distintos sistemas
func parseArpOutput(output string, ip string) (string, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, ip) {
			// en algunos sistemas la salida no contiene la IP en la línea con la MAC,
			// así que intentamos detectar tokens parecidos a MAC en cualquier línea.
			toks := strings.Fields(line)
			for _, t := range toks {
				clean := strings.Trim(t, "[],()")
				if looksLikeMAC(clean) {
					return strings.ToLower(strings.ReplaceAll(clean, "-", ":")), nil
				}
			}
			continue
		}
		// Si la línea contiene la IP, buscamos token mac en los tokens
		toks := strings.Fields(line)
		for _, t := range toks {
			clean := strings.Trim(t, "[],()")
			if looksLikeMAC(clean) {
				return strings.ToLower(strings.ReplaceAll(clean, "-", ":")), nil
			}
		}
		// caso macOS: "? (192.168.1.10) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]"
		if idx := strings.Index(line, " at "); idx != -1 {
			rest := line[idx+4:]
			parts := strings.Fields(rest)
			if len(parts) > 0 {
				clean := strings.Trim(parts[0], ",")
				if looksLikeMAC(clean) {
					return strings.ToLower(strings.ReplaceAll(clean, "-", ":")), nil
				}
			}
		}
	}
	return "", errors.New("MAC no encontrada en salida ARP")
}
