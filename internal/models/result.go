package models

type Result struct {
	IP         string `json:"ip"`
	Alive      bool   `json:"alive"`
	Method     string `json:"method,omitempty"`
	Port       int    `json:"port,omitempty"`
	MAC        string `json:"mac,omitempty"`
	ReverseDNS string `json:"reverse_dns,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
}
