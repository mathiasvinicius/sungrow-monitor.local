package inverter

// Sungrow Modbus Register Addresses
// Note: Modbus address = Register number - 1

const (
	// Device Information (Input Registers)
	RegSerialNumber     = 4989 // 4990-4999, String (10 registers)
	RegDeviceTypeCode   = 4999 // 5000, U16
	RegNominalPower     = 5000 // 5001, U16, 0.1kW
	RegOutputType       = 5001 // 5002, U16 (0=Single phase, 1=3P4L, 2=3P3L)

	// Production Data (Input Registers)
	RegDailyEnergy       = 5002 // 5003, U16, 0.1kWh
	RegTotalEnergy       = 5003 // 5004-5005, U32, 0.1kWh
	RegInsideTemperature = 5007 // 5008, S16, 0.1°C

	// MPPT Data
	RegMPPT1Voltage = 5010 // 5011, U16, 0.1V
	RegMPPT1Current = 5011 // 5012, U16, 0.01A
	RegMPPT2Voltage = 5012 // 5013, U16, 0.1V
	RegMPPT2Current = 5013 // 5014, U16, 0.01A
	RegTotalDCPower = 5016 // 5017-5018, U32, W

	// Grid Data
	RegPhaseAVoltage  = 5018 // 5019, U16, 0.1V
	RegPhaseBVoltage  = 5019 // 5020, U16, 0.1V
	RegPhaseCVoltage  = 5020 // 5021, U16, 0.1V
	RegGridFrequency  = 5021 // 5022, U16, 0.1Hz
	RegPhaseACurrent  = 5022 // 5023, U16, 0.1A
	RegPhaseBCurrent  = 5023 // 5024, U16, 0.1A
	RegPhaseCCurrent  = 5024 // 5025, U16, 0.1A

	// Power Data
	RegTotalActivePower   = 5030 // 5031-5032, U32, W
	RegReactivePower      = 5032 // 5033-5034, S32, var
	RegPowerFactor        = 5034 // 5035, S16, 0.001
	RegTotalApparentPower = 5035 // 5036-5037, U32, VA

	// Status
	RegRunningState   = 5037 // 5038, U16
	RegFaultCode      = 5039 // 5040, U16
	RegNominalReactivePower = 5048 // 5049, S16, 0.1kvar
)

// Running states
const (
	StateStop        = 0x0000
	StateStandby     = 0x8000
	StateStartup     = 0x1300
	StateMPPT        = 0x1400
	StateFault       = 0x1500
	StatePowerLimit  = 0x1600
	StateShutdown    = 0x1700
)

// Output types
const (
	OutputSinglePhase = 0
	Output3P4L        = 1
	Output3P3L        = 2
)

func GetRunningStateString(state uint16) string {
	switch state {
	case StateStop:
		return "Stop"
	case StateStandby:
		return "Standby"
	case StateStartup:
		return "Starting up"
	case StateMPPT:
		return "MPPT"
	case StateFault:
		return "Fault"
	case StatePowerLimit:
		return "Power limiting"
	case StateShutdown:
		return "Shutdown"
	default:
		return "Unknown"
	}
}

func GetOutputTypeString(outputType uint16) string {
	switch outputType {
	case OutputSinglePhase:
		return "Single Phase"
	case Output3P4L:
		return "Three Phase 4 Lines"
	case Output3P3L:
		return "Three Phase 3 Lines"
	default:
		return "Unknown"
	}
}
