package models

type Equipo struct {
	UUID        string      `json:"uuid"`
	MAC         string      `json:"mac"`
	Hostname    string      `json:"hostname"`
	OS          string      `json:"os"`
	OSVersion   string      `json:"os_version"`
	CPU         string      `json:"cpu"`
	RAM         string      `json:"ram"`
	Disk        []Disk      `json:"disk"`
	BIOS        BIOS        `json:"bios"`
	IP          string      `json:"ip"`
	Usuario     string      `json:"usuario"`
	AnyDesk     string      `json:"anydesk,omitempty"`
	RustDesk    string      `json:"rustdesk,omitempty"`
	Motherboard Motherboard `json:"motherboard"`
	GPU         string      `json:"gpu,omitempty"`
	CodActivo   string      `json:"cod_activo,omitempty"`
	Cargo       string      `json:"cargo,omitempty"`
	Unidad      string      `json:"unidad,omitempty"`
}

type Disk struct {
	Model string `json:"model"`
	Size  uint64 `json:"size"`
	Type  string `json:"type"`
}

type BIOS struct {
	Manufacturer string `json:"manufacturer"`
	Version      string `json:"version"`
	Date         string `json:"date"`
}

type Motherboard struct {
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
	SerialNumber string `json:"serial_number"`
}
