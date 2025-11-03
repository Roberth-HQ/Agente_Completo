package scan

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func ipsFromCIDR(ipnet *net.IPNet) []string {
	ip := ipnet.IP.To4()
	if ip == nil {
		return nil
	}
	var ips []string
	maskOnes, bits := ipnet.Mask.Size()
	if maskOnes == bits {
		return []string{ip.String()}
	}
	start := binaryIP(ip)
	hosts := 1 << (bits - maskOnes)
	for i := 0; i < hosts; i++ {
		ips = append(ips, ipFromUint32(start+uint32(i)).String())
	}
	return ips
}

func binaryIP(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8 | uint32(ip4[3])
}

func ipFromUint32(u uint32) net.IP {
	return net.IPv4(byte(u>>24), byte(u>>16), byte(u>>8), byte(u)).To4()
}

func ipsFromRange(a, b net.IP) ([]string, error) {
	ai := binaryIP(a.To4())
	bi := binaryIP(b.To4())
	if ai == 0 || bi == 0 {
		return nil, fmt.Errorf("solo IPv4 soportado")
	}
	if ai > bi {
		ai, bi = bi, ai
	}
	var out []string
	for v := ai; v <= bi; v++ {
		out = append(out, ipFromUint32(v).String())
	}
	return out, nil
}

// ----------------------- puertos y scanning -------------------------

func tryPing(ip string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// en windows -w espera ms; -n cantidad
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", fmt.Sprintf("%d", timeout.Milliseconds()), ip)
	} else {
		// en unix ping -c 1 (no usamos -W porque no está estandarizado en todos los sistemas)
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", ip)
	}
	err := cmd.Run()
	return err == nil
}

func tryTCP(ip string, port int, timeout time.Duration) bool {
	addr := fmt.Sprintf("%s:%d", ip, port)
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ----------------------- Device fingerprint heuristics -------------------------

// func detectDeviceType(ip string, ports []int, timeout time.Duration, mac, ptr string) string {
// 	has := func(p int) bool { return tryTCP(ip, p, timeout) }

// 	// Printer ports
// 	if has(9100) || has(631) || has(515) {
// 		h := probeHTTPForHints(ip, timeout)
// 		if h != "" {
// 			return "Printer (" + h + ")"
// 		}
// 		if looksLikePrinterName(ptr) {
// 			return "Printer (ptr)"
// 		}
// 		return "Printer"
// 	}

// 	// SMB/NAS
// 	if has(445) || has(139) {
// 		if strings.Contains(strings.ToLower(ptr), "nas") {
// 			return "NAS/SMB"
// 		}
// 		return "NAS/Windows SMB"
// 	}

// 	// RDP
// 	if has(3389) {
// 		return "Windows (RDP)"
// 	}

// 	// SSH
// 	if has(22) {
// 		if strings.Contains(strings.ToLower(ptr), "raspberry") || strings.Contains(strings.ToLower(ptr), "raspi") {
// 			return "Raspberry Pi (Linux/SSH)"
// 		}
// 		return "Linux/Unix (SSH)"
// 	}

// 	// Web-ish
// 	if has(80) || has(443) || has(8080) {
// 		h := probeHTTPForHints(ip, timeout)
// 		if h != "" {
// 			if looksLikePrinterHint(h) || looksLikePrinterName(ptr) {
// 				return "Printer (" + h + ")"
// 			}
// 			return "Web server (" + h + ")"
// 		}
// 		return "Web server"
// 	}

// 	// DB/IoT heuristics
// 	if has(3306) || has(5432) {
// 		return "DB server / Backend"
// 	}

// 	// fallback: use PTR content
// 	lptr := strings.ToLower(ptr)
// 	if lptr != "" {
// 		if looksLikePrinterName(ptr) {
// 			return "Printer (ptr)"
// 		}
// 		if strings.Contains(lptr, "router") || strings.Contains(lptr, "gateway") {
// 			return "Router/Gateway"
// 		}
// 		if strings.Contains(lptr, "camera") || strings.Contains(lptr, "ipcam") {
// 			return "IP Camera"
// 		}
// 	}

// 	// MAC vendor heuristics (muy limitado)
// 	if mac != "" {
// 		macLow := strings.ToLower(mac)
// 		if strings.HasPrefix(macLow, "00:1a:") || strings.HasPrefix(macLow, "00:1b:") {
// 			return "Device (MAC vendor hint)"
// 		}
// 	}

//		return "Unknown"
//	}
func detectDeviceType(ip string, ports []int, timeout time.Duration, mac string, reverseDNS string) string {
	// 1) si ya es impresora mantenlo (si tienes lógica de impresora en otro lado)
	// ejemplo simple: si puerto 9100 o 631 detecta impresora rápidamente
	if tryTCP(ip, 9100, timeout/2) || tryTCP(ip, 631, timeout/2) || tryTCP(ip, 515, timeout/2) {
		return "Printer"
	}

	// 2) OUI heurística
	vendor := simpleOUIVendor(mac)
	if strings.Contains(strings.ToLower(vendor), "camera") || strings.Contains(strings.ToLower(vendor), "dahua") {
		return "Camera"
	}
	if strings.Contains(strings.ToLower(vendor), "yealink") || strings.Contains(strings.ToLower(vendor), "polycom") {
		return "VoIP phone"
	}
	if strings.ToLower(vendor) == "apple" {
		// puede ser iPhone / Mac / iPad -> dejamos Unknown y lo intentamos con mDNS/http
	}

	// 3) quick port probes + banner heuristics
	// RTSP -> camera
	if tryTCP(ip, 554, timeout/2) {
		// intentar banner o DESCRIBE simple via TCP read
		ban := bannerProbe(ip, 554, timeout/2)
		if strings.Contains(strings.ToLower(ban), "rtsp") || strings.Contains(strings.ToLower(ban), "camera") {
			return "Camera"
		}
		return "Camera"
	}

	// SIP/VoIP -> telefono IP
	if tryTCP(ip, 5060, timeout/2) || tryTCP(ip, 5061, timeout/2) {
		return "VoIP phone"
	}

	// SMB/Netbios/RDP/SSH -> probablemente PC/Server
	if tryTCP(ip, 445, timeout/2) || tryTCP(ip, 139, timeout/2) || tryTCP(ip, 3389, timeout/2) || tryTCP(ip, 22, timeout/2) {
		return "PC"
	}

	// HTTP: chequear título / server para hints (cámaras y algunos móviles exponen admin pages)
	if tryTCP(ip, 80, timeout/2) || tryTCP(ip, 8080, timeout/2) || tryTCP(ip, 8000, timeout/2) {
		server, title := httpProbeTitle(ip, 80, timeout/2)
		lower := strings.ToLower(server + " " + title)
		if strings.Contains(lower, "hikvision") || strings.Contains(lower, "dahua") || strings.Contains(lower, "axis") {
			return "Camera"
		}
		if strings.Contains(lower, "android") || strings.Contains(lower, "iphone") || strings.Contains(lower, "apple") {
			return "Mobile"
		}
		if strings.Contains(lower, "phone") || strings.Contains(lower, "sip") || strings.Contains(lower, "asterisk") {
			return "VoIP phone"
		}
		// si no encontramos pistas, pero hay HTTP, puede ser PC/IoT -> marcar unknown para no false-positive
	}

	// 4) heurística por MAC -> mobiles / hubs
	if vendor == "Apple" {
		// si tiene HTTP y nombre via reverseDNS probablemente es "Mobile" o "PC"
		if reverseDNS != "" {
			if strings.Contains(strings.ToLower(reverseDNS), "iphone") || strings.Contains(strings.ToLower(reverseDNS), "ipad") {
				return "Mobile"
			}
			// macbook suele tener 'macbook' o 'mac' en reverse dns
			if strings.Contains(strings.ToLower(reverseDNS), "mac") {
				return "PC"
			}
		}
		// si nada, devolver Mobile como hipótesis baja
		return "Mobile"
	}

	// fallback
	return "Unknown"
}

// probeHTTPForHints intenta un HEAD/GET muy corto para obtener Server o title
func probeHTTPForHints(ip string, timeout time.Duration) string {
	try := func(port int) string {
		addr := fmt.Sprintf("%s:%d", ip, port)
		d := net.Dialer{Timeout: timeout}
		conn, err := d.Dial("tcp", addr)
		if err != nil {
			return ""
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		req := "HEAD / HTTP/1.0\r\nHost: " + ip + "\r\n\r\n"
		_, _ = conn.Write([]byte(req))
		r := bufio.NewReader(conn)
		var headers []string
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			headers = append(headers, line)
			if len(headers) > 50 {
				break
			}
		}
		for _, h := range headers {
			if strings.HasPrefix(strings.ToLower(h), "server:") {
				return strings.TrimSpace(h[len("server:"):])
			}
			if strings.HasPrefix(strings.ToLower(h), "title:") {
				return strings.TrimSpace(h[len("title:"):])
			}
		}
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		_, _ = conn.Write([]byte("GET / HTTP/1.0\r\nHost: " + ip + "\r\n\r\n"))
		bodyBuf := make([]byte, 2048)
		n, _ := r.Read(bodyBuf)
		body := strings.ToLower(string(bodyBuf[:n]))
		if idx := strings.Index(body, "<title>"); idx != -1 {
			end := strings.Index(body[idx:], "</title>")
			if end != -1 {
				title := strings.TrimSpace(body[idx+7 : idx+end])
				return title
			}
		}
		vendors := []string{"hp", "epson", "canon", "xerox", "konica", "kyocera", "brother"}
		for _, v := range vendors {
			if strings.Contains(body, v) {
				return v
			}
		}
		return ""
	}

	if s := try(80); s != "" {
		return s
	}
	if s := try(8080); s != "" {
		return s
	}
	if s := try(443); s != "" {
		return s
	}
	return ""
}

func looksLikePrinterHint(s string) bool {
	if s == "" {
		return false
	}
	l := strings.ToLower(s)
	return strings.Contains(l, "printer") || strings.Contains(l, "hp") || strings.Contains(l, "epson") || strings.Contains(l, "xerox") || strings.Contains(l, "printer")
}

func looksLikePrinterName(s string) bool {
	if s == "" {
		return false
	}
	l := strings.ToLower(s)
	keywords := []string{"printer", "hp", "epson", "canon", "brother", "lexmark", "ricoh", "xerox", "konica", "kyocera"}
	for _, k := range keywords {
		if strings.Contains(l, k) {
			return true
		}
	}
	return false
}

func ipLess(a, b string) bool {
	ai := binaryIP(net.ParseIP(a).To4())
	bi := binaryIP(net.ParseIP(b).To4())
	return ai < bi
}
