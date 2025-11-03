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

// getMACSimple intenta obtener la MAC para una IP usando:
// 1) ip neigh show <ip> (Linux) -> busca "lladdr"
// 2) /proc/net/arp (Linux)
// 3) arp -n <ip> o arp -a (fallback)
// Devuelve la MAC en minúsculas con ":" o error.
func getMAC(ip string, timeout time.Duration) (string, error) {
	// Forzar creación de entrada ARP (ping / arping)
	ensureARPEntrySimple(ip, timeout)

	// 1) ip neigh (Linux) — muy recomendable
	if runtime.GOOS == "linux" {
		if mac := macFromIPNeigh(ip); mac != "" {
			return mac, nil
		}
		// 2) /proc/net/arp
		if mac := readMACFromProcNetARP(ip); mac != "" {
			return mac, nil
		}
	}

	// 3) Fallback: comando arp (Windows / *nix)
	if mac := macFromARPCommandSimple(ip); mac != "" {
		return mac, nil
	}

	return "", errors.New("MAC no encontrada")
}

// ensureARPEntrySimple hace ping y, si está instalado, arping para poblar ARP.
func ensureARPEntrySimple(ip string, timeout time.Duration) {
	// Ping (cross-platform)
	pingTimeout := int(timeout.Seconds())
	if pingTimeout < 1 {
		pingTimeout = 1
	}

	if runtime.GOOS == "windows" {
		// Windows: ping -n 1 -w <ms>
		_ = exec.Command("ping", "-n", "1", "-w", fmt.Sprintf("%d", pingTimeout*1000), ip).Run()
	} else {
		// Unix: ping -c 1 -W <sec>
		_ = exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", pingTimeout), ip).Run()
	}

	// intentar arping si existe (solo Linux normalmente)
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("arping"); err == nil {
			// arping -c 1 -w <seg>
			_ = exec.Command("arping", "-c", "1", "-w", fmt.Sprintf("%d", pingTimeout), ip).Run()
		}
	}
}

// macFromIPNeigh usa `ip neigh show <ip>` y busca "lladdr <mac>"
func macFromIPNeigh(ip string) string {
	out, err := exec.Command("ip", "neigh", "show", ip).Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	s := string(out)
	// Ejemplo: "192.168.0.24 dev eth0 lladdr f8:63:d9:9b:93:a3 REACHABLE"
	if idx := strings.Index(s, "lladdr "); idx != -1 {
		rest := s[idx+7:]
		fields := strings.Fields(rest)
		if len(fields) > 0 && looksLikeMAC(fields[0]) {
			return strings.ToLower(normalizeMAC(fields[0]))
		}
	}
	// También puede venir sin 'lladdr' pero con 'REACHABLE' - buscamos token MAC
	for _, token := range strings.Fields(s) {
		if looksLikeMAC(token) {
			return strings.ToLower(normalizeMAC(token))
		}
	}
	return ""
}

// macFromARPCommandSimple intenta `arp -n <ip>` o `arp -a` y parsea salida minimal
func macFromARPCommandSimple(ip string) string {
	var out []byte
	var err error

	if runtime.GOOS == "windows" {
		out, err = exec.Command("arp", "-a", ip).CombinedOutput()
	} else {
		// en linux/mac intentamos `arp -n <ip>` o `arp -n`
		out, err = exec.Command("arp", "-n", ip).CombinedOutput()
		if err != nil || len(out) == 0 {
			out, _ = exec.Command("arp", "-n").CombinedOutput()
		}
	}
	if err != nil && len(out) == 0 {
		return ""
	}
	text := string(out)

	// Buscar token que parezca MAC en la línea que contenga la IP
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(line, ip) {
			for _, tok := range strings.Fields(line) {
				c := strings.Trim(tok, "[],()")
				if looksLikeMAC(c) {
					return strings.ToLower(normalizeMAC(c))
				}
			}
		}
	}

	// Si no hay línea con la IP, buscar cualquier MAC en salida (fallback)
	for _, tok := range strings.Fields(text) {
		c := strings.Trim(tok, "[],()")
		if looksLikeMAC(c) {
			return strings.ToLower(normalizeMAC(c))
		}
	}
	return ""
}

// normalizeMAC convierte formatos con "-" a ":" y limpia
func normalizeMAC(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), "-", ":")
}

// looksLikeMAC revisa patrón xx:xx:xx:xx:xx:xx (hex)
func looksLikeMAC(s string) bool {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", ":")
	parts := strings.Split(s, ":")
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
