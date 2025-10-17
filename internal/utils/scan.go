package scan

import (
	"escaner/internal/models"
	"fmt"
	"net"
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
func ScanIPs(ips []string, ports []int, timeout time.Duration, concurrency int) []models.Result {
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
			// ICMP
			if tryPing(ip, timeout) {
				res.Alive = true
				res.Method = "icmp"
			} else {
				// TCP fallback
				for _, p := range ports {
					if tryTCP(ip, p, timeout) {
						res.Alive = true
						res.Method = "tcp"
						res.Port = p
						break
					}
				}
			}

			// MAC: forzamos una acción que rellene la caché ARP y luego leemos ARP
			mac, _ := getMAC(ip, timeout)
			res.MAC = mac

			// Reverse DNS
			if names, err := net.LookupAddr(ip); err == nil && len(names) > 0 {
				res.ReverseDNS = strings.TrimSuffix(names[0], ".")
			}

			// Device detection
			res.DeviceType = detectDeviceType(ip, ports, timeout, res.MAC, res.ReverseDNS)

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
