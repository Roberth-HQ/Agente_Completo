package scan

import (
	"bufio"
	"escaner/internal/models"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ----------------------- utilidad de parsing y rangos -------------------------

func ExpandArgToIPs(arg string) ([]string, error) {
	arg = strings.TrimSpace(arg)
	// CIDR
	if strings.Contains(arg, "/") {
		_, ipnet, err := net.ParseCIDR(arg)
		if err != nil {
			return nil, err
		}
		return ipsFromCIDR(ipnet), nil
	}
	// shorthand last-octet e.g., 192.168.181.1-255
	if strings.Count(arg, "-") == 1 && !strings.Contains(arg, " ") {
		parts := strings.Split(arg, "-")
		left := parts[0]
		right := parts[1]
		if net.ParseIP(left) != nil {
			ip4 := net.ParseIP(left).To4()
			if ip4 != nil {
				if hi, err := strconv.Atoi(right); err == nil {
					if hi >= 0 && hi <= 255 {
						base := fmt.Sprintf("%d.%d.%d", ip4[0], ip4[1], ip4[2])
						lo := int(ip4[3])
						if lo > hi {
							lo, hi = hi, lo
						}
						out := make([]string, 0, hi-lo+1)
						for i := lo; i <= hi; i++ {
							out = append(out, fmt.Sprintf("%s.%d", base, i))
						}
						return out, nil
					}
				}
			}
		}
		// fallthrough to full-range parsing
	}
	// full range: 192.168.1.10-192.168.1.50
	if strings.Count(arg, "-") == 1 {
		parts := strings.Split(arg, "-")
		a := net.ParseIP(strings.TrimSpace(parts[0]))
		b := net.ParseIP(strings.TrimSpace(parts[1]))
		if a == nil || b == nil {
			return nil, fmt.Errorf("rango ip inválido")
		}
		return ipsFromRange(a, b)
	}
	// single IP
	if net.ParseIP(arg) != nil {
		return []string{arg}, nil
	}
	return nil, fmt.Errorf("formato de IP no reconocido")
}

// ----------------------- función reutilizable de escaneo -------------------------

// scanIPs realiza el escaneo paralelo de la lista de IPs usando las mismas heurísticas
func ScanIPs(
	ips []string,
	ports []int,
	timeout time.Duration,
	concurrency int,
	onAlive func(models.Result), // nuevo parámetro
) []models.Result {
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	resultsCh := make(chan models.Result, len(ips))

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()

			res := models.Result{IP: ip}
			if tryPing(ip, timeout) {
				res.Alive = true
				res.Method = "icmp"
			} else {
				for _, p := range ports {
					if tryTCP(ip, p, timeout) {
						res.Alive = true
						res.Method = "tcp"
						res.Port = p
						break
					}
				}
			}

			mac, _ := getMAC(ip, timeout)
			res.MAC = mac
			if names, err := net.LookupAddr(ip); err == nil && len(names) > 0 {
				res.ReverseDNS = strings.TrimSuffix(names[0], ".")
			}

			// primero detectar tipo (usa reverseDNS y MAC)
			res.DeviceType = detectDeviceType(ip, ports, timeout, res.MAC, res.ReverseDNS)
			if res.Alive && res.DeviceType == "Unknown" {
				if brand, ok := isMobileOUI(res.MAC); ok {
					res.DeviceType = "Mobile"
					if res.ReverseDNS == "" {
						res.ReverseDNS = brand + " Mobile"
					}
				}
			}

			// luego enriquecer name: si no hay reverseDNS intentamos HTTP/banner heuristics
			res.ReverseDNS = enrichName(ip, timeout, res.ReverseDNS)

			// Llamamos al callback si está vivo
			if res.Alive && onAlive != nil {
				onAlive(res)
			}

			resultsCh <- res
		}(ip)
	}

	wg.Wait()
	close(resultsCh)

	var results []models.Result
	for r := range resultsCh {
		results = append(results, r)
	}
	sort.Slice(results, func(i, j int) bool {
		return ipLess(results[i].IP, results[j].IP)
	})

	return results
}

// /saber mobiles:
var mobileOUIs = map[string]string{
	"c2:a1:6f": "Apple",
	"d4:61:9d": "Samsung",
	"f0:27:2d": "Xiaomi",
	"44:65:0d": "Huawei",
	"ac:bc:32": "Motorola",
	"3486da":   "Honor",
	// puedes agregar más aquí
}

func isMobileOUI(mac string) (string, bool) {
	if mac == "" {
		return "", false
	}
	// normalizar: quitar separadores y pasar a minúsculas
	cleaned := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(mac, ":", ""), "-", ""))
	if len(cleaned) < 6 {
		return "", false
	}
	prefix := cleaned[:6] // primeros 3 bytes
	if brand, ok := mobileOUIs[prefix]; ok {
		return brand, true
	}
	return "", false
}

// ----------------------- formato de salida -------------------------

func FormatResult(r models.Result) string {
	alive := "no"
	method := "-"
	if r.Alive {
		alive = "sí"
		if r.Method == "icmp" {
			method = "ICMP"
		} else if r.Method == "tcp" {
			method = fmt.Sprintf("TCP/%d", r.Port)
		}
	}
	mac := r.MAC
	if mac == "" {
		mac = "-"
	}
	name := r.ReverseDNS
	if name == "" {
		name = "-"
	}
	dev := r.DeviceType
	if dev == "" {
		dev = "-"
	}
	return fmt.Sprintf("%-15s  alive:%-3s  via:%-10s  device:%-20s  mac:%-17s  name:%s",
		r.IP, alive, method, dev, mac, name)
}

// ----------------------- puertos y scanning -------------------------

func ParsePorts(s string) []int {
	out := []int{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, err := strconv.Atoi(p); err == nil && v > 0 && v <= 65535 {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return []int{22, 80, 443}
	}
	return out
}

// ////mejoramiento
func simpleOUIVendor(mac string) string {
	m := strings.ToLower(strings.ReplaceAll(mac, ":", ""))
	if len(m) < 6 {
		return ""
	}
	prefix := m[:6]
	switch prefix {
	case "00163e", "001e65", "f8f63d": // ejemplos Apple / Samsung / sospechosos (ajusta a tu OUI)
		return "Apple"
	case "34b3f6", "34a4f9":
		return "Samsung"
	case "3c5a37", "3c5a47":
		return "Dahua/CameraVendor" // ejemplo
	case "001d1a", "001ec0":
		return "Hikvision/Camera"
	case "a0b1c2":
		return "Yealink/VoIP"
	case "000b82":
		return "Grandstream/VoIP"

	}
	return ""
}

// bannerProbe: intenta conectar TCP y leer primeros bytes (banner)
func bannerProbe(ip string, port int, timeout time.Duration) string {
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return ""
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	return string(buf[:n])
}

// httpProbeTitle: hace GET / y devuelve header Server o <title> de la página
func httpProbeTitle(ip string, port int, timeout time.Duration) (string, string) {
	client := &http.Client{Timeout: timeout}
	url := fmt.Sprintf("http://%s:%d/", ip, port)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (scan)")
	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	server := resp.Header.Get("Server")
	// leer los primeros KB y buscar <title>
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
	title := ""
	re := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	count := 0
	for scanner.Scan() && count < 20 {
		line := scanner.Text()
		if match := re.FindStringSubmatch(line); len(match) > 1 {
			title = strings.TrimSpace(match[1])
			break
		}
		count++
	}
	return server, title
}

// enrichName intenta obtener mejor nombre/modelo: reverseDNS -> http Server/Title -> banner
func enrichName(ip string, timeout time.Duration, currentName string) string {
	if currentName != "" {
		return currentName
	}
	// 1) intentar http title/server
	if tryTCP(ip, 80, timeout/2) {
		server, title := httpProbeTitle(ip, 80, timeout/2)
		if title != "" {
			return title
		}
		if server != "" {
			return server
		}
	}
	// 2) banner probe common ports for model hints
	ports := []int{554, 22, 80, 8080, 8000}
	for _, p := range ports {
		if tryTCP(ip, p, timeout/2) {
			b := bannerProbe(ip, p, timeout/2)
			// buscar patrones comunes
			bL := strings.ToLower(b)
			if strings.Contains(bL, "hikvision") || strings.Contains(bL, "dahua") {
				return strings.TrimSpace(firstMatch(b, `(?i)(hikvision|dahua)[^\s<>]{0,40}`))
			}
			if strings.Contains(bL, "iphone") || strings.Contains(bL, "android") {
				return firstMatch(b, `(?i)(iphone|android[^\s<>]*)`)
			}
			if len(b) > 0 {
				// recortar banner razonable
				s := strings.TrimSpace(b)
				if len(s) > 60 {
					s = s[:60]
				}
				return s
			}
		}
	}
	return ""
}

// firstMatch devuelve el primer grupo que casé con regex, o "" si no hay match
func firstMatch(s string, pattern string) string {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(s)
	if len(m) > 0 {
		return strings.TrimSpace(m[0])
	}
	return ""
}
