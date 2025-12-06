package inverter

import (
	"fmt"
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

	// Grid
	PhaseAVoltage float64 `json:"phase_a_voltage_v"`
	PhaseBVoltage float64 `json:"phase_b_voltage_v"`
	PhaseCVoltage float64 `json:"phase_c_voltage_v"`
	GridFrequency float64 `json:"grid_frequency_hz"`
	PhaseACurrent float64 `json:"phase_a_current_a"`
	PhaseBCurrent float64 `json:"phase_b_current_a"`
	PhaseCCurrent float64 `json:"phase_c_current_a"`

	// Power
	TotalActivePower   uint32  `json:"total_active_power_w"`
	ReactivePower      int32   `json:"reactive_power_var"`
	PowerFactor        float64 `json:"power_factor"`
	TotalApparentPower uint32  `json:"total_apparent_power_va"`

	// Status
	RunningState       uint16 `json:"running_state"`
	RunningStateString string `json:"running_state_string"`
	FaultCode          uint16 `json:"fault_code"`
	IsOnline           bool   `json:"is_online"`
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
		IsOnline:  true,
	}

	// Read device info
	serial, err := s.client.ReadString(RegSerialNumber, 10)
	if err != nil {
		data.IsOnline = false
		return data, fmt.Errorf("failed to read serial number: %w", err)
	}
	data.SerialNumber = serial

	deviceType, err := s.client.ReadUint16(RegDeviceTypeCode)
	if err != nil {
		return data, fmt.Errorf("failed to read device type: %w", err)
	}
	data.DeviceTypeCode = deviceType

	nominalPower, err := s.client.ReadUint16(RegNominalPower)
	if err != nil {
		return data, fmt.Errorf("failed to read nominal power: %w", err)
	}
	data.NominalPower = float64(nominalPower) * 0.1

	outputType, err := s.client.ReadUint16(RegOutputType)
	if err != nil {
		return data, fmt.Errorf("failed to read output type: %w", err)
	}
	data.OutputType = GetOutputTypeString(outputType)

	// Read energy data
	dailyEnergy, err := s.client.ReadUint16(RegDailyEnergy)
	if err != nil {
		return data, fmt.Errorf("failed to read daily energy: %w", err)
	}
	data.DailyEnergy = float64(dailyEnergy) * 0.1

	totalEnergy, err := s.client.ReadUint32(RegTotalEnergy)
	if err != nil {
		return data, fmt.Errorf("failed to read total energy: %w", err)
	}
	data.TotalEnergy = float64(totalEnergy) * 0.1

	// Read temperature
	temp, err := s.client.ReadInt16(RegInsideTemperature)
	if err != nil {
		return data, fmt.Errorf("failed to read temperature: %w", err)
	}
	data.Temperature = float64(temp) * 0.1

	// Read MPPT data
	mppt1v, err := s.client.ReadUint16(RegMPPT1Voltage)
	if err != nil {
		return data, fmt.Errorf("failed to read MPPT1 voltage: %w", err)
	}
	data.MPPT1Voltage = float64(mppt1v) * 0.1

	mppt1c, err := s.client.ReadUint16(RegMPPT1Current)
	if err != nil {
		return data, fmt.Errorf("failed to read MPPT1 current: %w", err)
	}
	data.MPPT1Current = float64(mppt1c) * 0.01

	mppt2v, err := s.client.ReadUint16(RegMPPT2Voltage)
	if err != nil {
		return data, fmt.Errorf("failed to read MPPT2 voltage: %w", err)
	}
	data.MPPT2Voltage = float64(mppt2v) * 0.1

	mppt2c, err := s.client.ReadUint16(RegMPPT2Current)
	if err != nil {
		return data, fmt.Errorf("failed to read MPPT2 current: %w", err)
	}
	data.MPPT2Current = float64(mppt2c) * 0.01

	dcPower, err := s.client.ReadUint32(RegTotalDCPower)
	if err != nil {
		return data, fmt.Errorf("failed to read DC power: %w", err)
	}
	data.TotalDCPower = dcPower

	// Read grid data
	phaseAv, err := s.client.ReadUint16(RegPhaseAVoltage)
	if err != nil {
		return data, fmt.Errorf("failed to read phase A voltage: %w", err)
	}
	data.PhaseAVoltage = float64(phaseAv) * 0.1

	phaseBv, err := s.client.ReadUint16(RegPhaseBVoltage)
	if err != nil {
		return data, fmt.Errorf("failed to read phase B voltage: %w", err)
	}
	data.PhaseBVoltage = float64(phaseBv) * 0.1

	phaseCv, err := s.client.ReadUint16(RegPhaseCVoltage)
	if err != nil {
		return data, fmt.Errorf("failed to read phase C voltage: %w", err)
	}
	data.PhaseCVoltage = float64(phaseCv) * 0.1

	freq, err := s.client.ReadUint16(RegGridFrequency)
	if err != nil {
		return data, fmt.Errorf("failed to read grid frequency: %w", err)
	}
	data.GridFrequency = float64(freq) * 0.1

	phaseAc, err := s.client.ReadUint16(RegPhaseACurrent)
	if err != nil {
		return data, fmt.Errorf("failed to read phase A current: %w", err)
	}
	data.PhaseACurrent = float64(phaseAc) * 0.1

	phaseBc, err := s.client.ReadUint16(RegPhaseBCurrent)
	if err != nil {
		return data, fmt.Errorf("failed to read phase B current: %w", err)
	}
	data.PhaseBCurrent = float64(phaseBc) * 0.1

	phaseCc, err := s.client.ReadUint16(RegPhaseCCurrent)
	if err != nil {
		return data, fmt.Errorf("failed to read phase C current: %w", err)
	}
	data.PhaseCCurrent = float64(phaseCc) * 0.1

	// Read power data
	activePower, err := s.client.ReadUint32(RegTotalActivePower)
	if err != nil {
		return data, fmt.Errorf("failed to read active power: %w", err)
	}
	data.TotalActivePower = activePower

	reactivePower, err := s.client.ReadInt32(RegReactivePower)
	if err != nil {
		return data, fmt.Errorf("failed to read reactive power: %w", err)
	}
	data.ReactivePower = reactivePower

	pf, err := s.client.ReadInt16(RegPowerFactor)
	if err != nil {
		return data, fmt.Errorf("failed to read power factor: %w", err)
	}
	data.PowerFactor = float64(pf) * 0.001

	apparentPower, err := s.client.ReadUint32(RegTotalApparentPower)
	if err != nil {
		return data, fmt.Errorf("failed to read apparent power: %w", err)
	}
	data.TotalApparentPower = apparentPower

	// Read status
	state, err := s.client.ReadUint16(RegRunningState)
	if err != nil {
		return data, fmt.Errorf("failed to read running state: %w", err)
	}
	data.RunningState = state
	data.RunningStateString = GetRunningStateString(state)

	faultCode, err := s.client.ReadUint16(RegFaultCode)
	if err != nil {
		return data, fmt.Errorf("failed to read fault code: %w", err)
	}
	data.FaultCode = faultCode

	return data, nil
}

func (s *Sungrow) TestConnection() error {
	if err := s.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Try to read serial number as a connection test
	_, err := s.client.ReadString(RegSerialNumber, 10)
	if err != nil {
		return fmt.Errorf("failed to read from inverter: %w", err)
	}

	return nil
}
