package inverter

import (
	"log"
	"time"

	"sungrow-monitor/internal/modbus"
)

type InverterData struct {
	Timestamp time.Time `json:"timestamp"`

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

	// Grid (single phase for SG5.0RS-S)
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
	Errors             []string `json:"errors,omitempty"`
}

type Sungrow struct {
	client *modbus.Client
}

func NewSungrow(client *modbus.Client) *Sungrow {
	return &Sungrow{client: client}
}

func (s *Sungrow) ReadAllData() (*InverterData, error) {
	data := &InverterData{
		Timestamp: time.Now(),
		IsOnline:  false,
		Errors:    make([]string, 0),
	}

	// Try to read device info first - this is the connectivity test
	serial, err := s.client.ReadString(RegSerialNumber, 10)
	if err != nil {
		log.Printf("Failed to read serial (inverter may be offline): %v", err)
		return data, err
	}
	data.SerialNumber = serial
	data.IsOnline = true

	// Read device type
	if deviceType, err := s.client.ReadUint16(RegDeviceTypeCode); err == nil {
		data.DeviceTypeCode = deviceType
	} else {
		data.Errors = append(data.Errors, "device_type")
	}

	// Read nominal power
	if nominalPower, err := s.client.ReadUint16(RegNominalPower); err == nil {
		data.NominalPower = float64(nominalPower) * 0.1
	} else {
		data.Errors = append(data.Errors, "nominal_power")
	}

	// Read output type
	if outputType, err := s.client.ReadUint16(RegOutputType); err == nil {
		data.OutputType = GetOutputTypeString(outputType)
	} else {
		data.OutputType = "Single Phase" // Default for SG5.0RS-S
	}

	// Read energy data
	if dailyEnergy, err := s.client.ReadUint16(RegDailyEnergy); err == nil {
		data.DailyEnergy = float64(dailyEnergy) * 0.1
	} else {
		data.Errors = append(data.Errors, "daily_energy")
	}

	if totalEnergy, err := s.client.ReadUint32(RegTotalEnergy); err == nil {
		data.TotalEnergy = float64(totalEnergy) * 0.1
	} else {
		data.Errors = append(data.Errors, "total_energy")
	}

	// Read temperature
	if temp, err := s.client.ReadInt16(RegInsideTemperature); err == nil {
		data.Temperature = float64(temp) * 0.1
	} else {
		data.Errors = append(data.Errors, "temperature")
	}

	// Read MPPT1 data
	if mppt1v, err := s.client.ReadUint16(RegMPPT1Voltage); err == nil {
		data.MPPT1Voltage = float64(mppt1v) * 0.1
	}

	if mppt1c, err := s.client.ReadUint16(RegMPPT1Current); err == nil {
		data.MPPT1Current = float64(mppt1c) * 0.01
	}

	// Read MPPT2 data (may not exist on all models)
	if mppt2v, err := s.client.ReadUint16(RegMPPT2Voltage); err == nil {
		data.MPPT2Voltage = float64(mppt2v) * 0.1
	}

	if mppt2c, err := s.client.ReadUint16(RegMPPT2Current); err == nil {
		data.MPPT2Current = float64(mppt2c) * 0.01
	}

	// Read DC power
	if dcPower, err := s.client.ReadUint32(RegTotalDCPower); err == nil {
		data.TotalDCPower = dcPower
	}

	// Read grid data (single phase only for SG5.0RS-S)
	if gridV, err := s.client.ReadUint16(RegPhaseAVoltage); err == nil {
		data.GridVoltage = float64(gridV) * 0.1
	}

	if freq, err := s.client.ReadUint16(RegGridFrequency); err == nil {
		data.GridFrequency = float64(freq) * 0.1
	}

	if gridC, err := s.client.ReadUint16(RegPhaseACurrent); err == nil {
		data.GridCurrent = float64(gridC) * 0.1
	}

	// Read power data
	if activePower, err := s.client.ReadUint32(RegTotalActivePower); err == nil {
		data.TotalActivePower = activePower
	}

	if reactivePower, err := s.client.ReadInt32(RegReactivePower); err == nil {
		data.ReactivePower = reactivePower
	}

	if pf, err := s.client.ReadInt16(RegPowerFactor); err == nil {
		data.PowerFactor = float64(pf) * 0.001
	}

	// Read status
	if state, err := s.client.ReadUint16(RegRunningState); err == nil {
		data.RunningState = state
		data.RunningStateString = GetRunningStateString(state)
	} else {
		data.RunningStateString = "Unknown"
	}

	if faultCode, err := s.client.ReadUint16(RegFaultCode); err == nil {
		data.FaultCode = faultCode
	}

	return data, nil
}

func (s *Sungrow) TestConnection() error {
	if err := s.client.Connect(); err != nil {
		return err
	}

	// Try to read device type as a simple test
	_, err := s.client.ReadUint16(RegDeviceTypeCode)
	return err
}
