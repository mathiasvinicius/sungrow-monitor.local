package storage

import (
	"time"

	"gorm.io/gorm"
)

type InverterReading struct {
	gorm.Model
	Timestamp time.Time `gorm:"index" json:"timestamp"`

	// Device Info
	SerialNumber   string  `json:"serial_number"`
	DeviceTypeCode uint16  `json:"device_type_code"`
	NominalPower   float64 `json:"nominal_power_kw"`
	OutputType     string  `json:"output_type"`

	// Energy
	DailyEnergy float64 `json:"daily_energy_kwh"`
	TotalEnergy float64 `json:"total_energy_kwh"`

	// Temperature
	Temperature float64 `json:"temperature_c"`

	// MPPT
	MPPT1Voltage float64 `json:"mppt1_voltage_v"`
	MPPT1Current float64 `json:"mppt1_current_a"`
	MPPT2Voltage float64 `json:"mppt2_voltage_v"`
	MPPT2Current float64 `json:"mppt2_current_a"`
	TotalDCPower uint32  `json:"total_dc_power_w"`

	// Grid (single phase)
	GridVoltage   float64 `json:"grid_voltage_v"`
	GridFrequency float64 `json:"grid_frequency_hz"`
	GridCurrent   float64 `json:"grid_current_a"`

	// Power
	TotalActivePower uint32  `json:"total_active_power_w"`
	ReactivePower    int32   `json:"reactive_power_var"`
	PowerFactor      float64 `json:"power_factor"`

	// Status
	RunningState       uint16 `json:"running_state"`
	RunningStateString string `json:"running_state_string"`
	FaultCode          uint16 `json:"fault_code"`
	IsOnline           bool   `json:"is_online"`
}

type DailyStats struct {
	Date           time.Time `json:"date"`
	MaxPower       uint32    `json:"max_power_w"`
	TotalEnergy    float64   `json:"total_energy_kwh"`
	AvgTemperature float64   `json:"avg_temperature_c"`
	ReadingsCount  int64     `json:"readings_count"`
}
