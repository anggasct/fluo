package main

type SecurityMode string

const (
	Disarmed  SecurityMode = "disarmed"
	ArmedStay SecurityMode = "armed_stay"
	ArmedAway SecurityMode = "armed_away"
)

type ClimateMode string

const (
	ClimateOff ClimateMode = "off"
	Heating    ClimateMode = "heating"
	Cooling    ClimateMode = "cooling"
	Auto       ClimateMode = "auto"
)

type LightingMode string

const (
	Manual       LightingMode = "manual"
	Schedule     LightingMode = "schedule"
	MotionDetect LightingMode = "motion_detect"
)

type SmartHome struct {
	SecurityMode    SecurityMode
	ClimateMode     ClimateMode
	LightingMode    LightingMode
	Temperature     int
	IsOccupied      bool
	PowerSave       bool
	EmergencyActive bool
}

type SensorData struct {
	Temperature    int
	MotionDetected bool
	DoorOpen       bool
	WindowOpen     bool
	PowerLevel     int
}
