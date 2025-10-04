package main

import (
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

type SmartHomeObserver struct{ fluo.BaseObserver }

func (o *SmartHomeObserver) OnTransition(from, to string, event fluo.Event, ctx fluo.Context) {
	name := ""
	if event != nil {
		name = event.GetName()
	}
	fmt.Printf("%s --[%s]--> %s\n", from, name, to)
}

func (o *SmartHomeObserver) OnStateEnter(state string, ctx fluo.Context) {
	fmt.Printf("Entered: %s\n", state)
}

func main() {
	def := BuildSmartHomeMachine()

	fmt.Println("=== Scenario 1: Normal Day Cycle ===")
	m1 := def.CreateInstance()
	m1.AddObserver(&SmartHomeObserver{})
	_ = m1.Start()
	smartHome := &SmartHome{Temperature: 72, IsOccupied: true}
	sensorData := &SensorData{Temperature: 72, MotionDetected: false, DoorOpen: false, PowerLevel: 100}
	m1.Context().Set("smart_home", smartHome)
	m1.Context().Set("sensor_data", sensorData)
	runNormalDayScenario(m1, sensorData)

	fmt.Println("\n=== Scenario 2: Away Mode ===")
	m2 := def.CreateInstance()
	m2.AddObserver(&SmartHomeObserver{})
	_ = m2.Start()
	smartHome = &SmartHome{Temperature: 72, IsOccupied: false, PowerSave: true}
	sensorData = &SensorData{Temperature: 72, MotionDetected: false, DoorOpen: false, PowerLevel: 80}
	m2.Context().Set("smart_home", smartHome)
	m2.Context().Set("sensor_data", sensorData)
	runAwayModeScenario(m2, sensorData)

	fmt.Println("\n=== Scenario 3: Security Breach Emergency ===")
	m3 := def.CreateInstance()
	m3.AddObserver(&SmartHomeObserver{})
	_ = m3.Start()
	smartHome = &SmartHome{Temperature: 70, IsOccupied: false}
	sensorData = &SensorData{Temperature: 70, MotionDetected: false, DoorOpen: false, PowerLevel: 95}
	m3.Context().Set("smart_home", smartHome)
	m3.Context().Set("sensor_data", sensorData)
	runSecurityBreachScenario(m3, sensorData)
}

func runNormalDayScenario(m fluo.Machine, sensorData *SensorData) {
	steps := []struct {
		event      string
		desc       string
		tempChange func()
	}{
		{"power_on", "Power on the system", nil},
		{"system_ready", "System initialization complete", nil},
		{"arm_stay", "Arm security in stay mode", nil},
		{"climate_on", "Enable climate control", nil},
		{"schedule_mode", "Switch to scheduled lighting", nil},
		{"temp_high", "Temperature rises", func() { sensorData.Temperature = 78 }},
		{"temp_low", "Temperature drops", func() { sensorData.Temperature = 65 }},
		{"manual_mode", "Switch back to manual lighting", nil},
	}

	for i, step := range steps {
		fmt.Printf("\n-- Step %d: %s ('%s') --\n", i+1, step.desc, step.event)
		if step.tempChange != nil {
			step.tempChange()
		}
		res := m.SendEvent(step.event, nil)
		if !res.Success() {
			fmt.Println("rejected:", res.RejectionReason)
		}
		fmt.Println("Active:", m.GetActiveStates())
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Println("Current state:", m.CurrentState())
}

func runAwayModeScenario(m fluo.Machine, sensorData *SensorData) {
	steps := []struct {
		event string
		desc  string
		setup func()
	}{
		{"power_on", "Power on the system", nil},
		{"system_ready", "System ready", nil},
		{"arm_away", "Arm security in away mode", nil},
		{"climate_on", "Enable energy-efficient climate", nil},
		{"motion_mode", "Enable motion-based lighting", nil},
		{"temp_high", "Temperature adjustment needed", func() { sensorData.Temperature = 80 }},
	}

	for i, step := range steps {
		fmt.Printf("\n-- Step %d: %s ('%s') --\n", i+1, step.desc, step.event)
		if step.setup != nil {
			step.setup()
		}
		res := m.SendEvent(step.event, nil)
		if !res.Success() {
			fmt.Println("rejected:", res.RejectionReason)
		}
		fmt.Println("Active:", m.GetActiveStates())
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Println("Current state:", m.CurrentState())
}

func runSecurityBreachScenario(m fluo.Machine, sensorData *SensorData) {
	steps := []struct {
		event string
		desc  string
		setup func()
	}{
		{"power_on", "Power on the system", nil},
		{"system_ready", "System ready", nil},
		{"arm_away", "Arm security in away mode", nil},
		{"climate_on", "Enable climate control", nil},
		{"motion_detected", "Motion detected while armed away", func() {
			sensorData.MotionDetected = true
		}},
		{"emergency_clear", "Emergency situation resolved", func() {
			sensorData.MotionDetected = false
			m.Context().Set("emergency_clear", true)
		}},
	}

	for i, step := range steps {
		fmt.Printf("\n-- Step %d: %s ('%s') --\n", i+1, step.desc, step.event)
		if step.setup != nil {
			step.setup()
		}
		res := m.SendEvent(step.event, nil)
		if !res.Success() {
			fmt.Println("rejected:", res.RejectionReason)
		}
		fmt.Println("Active:", m.GetActiveStates())
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Println("Current state:", m.CurrentState())
}
