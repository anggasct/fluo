package main

import (
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

type TrafficLightObserver struct{ fluo.BaseObserver }

func (o *TrafficLightObserver) OnTransition(from, to string, event fluo.Event, ctx fluo.Context) {
	name := ""
	if event != nil {
		name = event.GetName()
	}
	fmt.Printf("%s --[%s]--> %s\n", from, name, to)
}

func (o *TrafficLightObserver) OnStateEnter(state string, ctx fluo.Context) {
	fmt.Printf("Entered: %s\n", state)
}

func main() {
	def := BuildTrafficLightMachine()

	fmt.Println("=== Scenario 1: Normal Traffic Light Cycle ===")
	runNormalOperation(def)

	fmt.Println("\n=== Scenario 2: Emergency Vehicle Override ===")
	runEmergencyScenario(def)

	fmt.Println("\n=== Scenario 3: Pedestrian Crossing ===")
	runPedestrianScenario(def)

	fmt.Println("\n=== Scenario 4: Maintenance Mode ===")
	runMaintenanceScenario(def)
}

func runNormalOperation(def fluo.MachineDefinition) {
	m := def.CreateInstance()
	m.AddObserver(&TrafficLightObserver{})

	intersection := &Intersection{
		ID:         "MAIN-001",
		NorthSouth: TrafficLight{ID: "NS-001", Direction: NorthSouth, Color: Red},
		EastWest:   TrafficLight{ID: "EW-001", Direction: EastWest, Color: Green},
	}

	_ = m.Start()
	m.Context().Set("intersection", intersection)

	steps := []struct {
		event       string
		description string
		waitTime    time.Duration
		setup       func()
	}{
		{"power_on", "Power on traffic light system", 100 * time.Millisecond, nil},
		{"system_ready", "System initialization complete", 100 * time.Millisecond, nil},
		{"timer_expired", "North-South red timer expired", 500 * time.Millisecond, func() {
			m.Context().Set("ns_timer", time.Now().Add(-1*time.Second))
		}},
		{"timer_expired", "North-South red+yellow timer expired", 500 * time.Millisecond, func() {
			m.Context().Set("ns_timer", time.Now().Add(-1*time.Second))
		}},
		{"timer_expired", "East-West green timer expired", 500 * time.Millisecond, func() {
			m.Context().Set("ew_timer", time.Now().Add(-1*time.Second))
		}},
		{"timer_expired", "East-West yellow timer expired", 500 * time.Millisecond, func() {
			m.Context().Set("ew_timer", time.Now().Add(-1*time.Second))
		}},
	}

	for i, step := range steps {
		fmt.Printf("\n-- Step %d: %s ('%s') --\n", i+1, step.description, step.event)
		if step.setup != nil {
			step.setup()
		}
		res := m.SendEvent(step.event, nil)
		if !res.Success() {
			fmt.Println("rejected:", res.RejectionReason)
		}
		fmt.Printf("Active: %v\n", m.GetActiveStates())
		fmt.Printf("Lights: NS=%s, EW=%s\n", intersection.NorthSouth.Color, intersection.EastWest.Color)
		time.Sleep(step.waitTime)
	}
}

func runEmergencyScenario(def fluo.MachineDefinition) {
	m := def.CreateInstance()
	m.AddObserver(&TrafficLightObserver{})

	intersection := &Intersection{
		ID:         "MAIN-001",
		NorthSouth: TrafficLight{ID: "NS-001", Direction: NorthSouth, Color: Green},
		EastWest:   TrafficLight{ID: "EW-001", Direction: EastWest, Color: Red},
	}

	_ = m.Start()
	m.Context().Set("intersection", intersection)

	steps := []struct {
		event       string
		description string
		setup       func()
	}{
		{"power_on", "Power on system", nil},
		{"system_ready", "System ready", nil},
		{"emergency_vehicle", "Emergency vehicle approaching!", func() {
			fmt.Println("  🚨 AMBULANCE DETECTED!")
		}},
		{"emergency_clear", "Emergency vehicle passed", func() {
			m.Context().Set("emergency_clear", true)
			fmt.Println("  ✅ Emergency vehicle cleared intersection")
		}},
	}

	for i, step := range steps {
		fmt.Printf("\n-- Step %d: %s ('%s') --\n", i+1, step.description, step.event)
		if step.setup != nil {
			step.setup()
		}
		res := m.SendEvent(step.event, nil)
		if !res.Success() {
			fmt.Println("rejected:", res.RejectionReason)
		}
		fmt.Printf("Active: %v\n", m.GetActiveStates())
		fmt.Printf("Lights: NS=%s, EW=%s\n", intersection.NorthSouth.Color, intersection.EastWest.Color)
		time.Sleep(200 * time.Millisecond)
	}
}

func runPedestrianScenario(def fluo.MachineDefinition) {
	m := def.CreateInstance()
	m.AddObserver(&TrafficLightObserver{})

	intersection := &Intersection{
		ID:         "MAIN-001",
		NorthSouth: TrafficLight{ID: "NS-001", Direction: NorthSouth, Color: Red},
		EastWest:   TrafficLight{ID: "EW-001", Direction: EastWest, Color: Green},
	}

	_ = m.Start()
	m.Context().Set("intersection", intersection)

	steps := []struct {
		event       string
		description string
		setup       func()
	}{
		{"power_on", "Power on system", nil},
		{"system_ready", "System ready", nil},
		{"pedestrian_button", "Pedestrian crossing button pressed", func() {
			fmt.Println("  🚶 Pedestrian waiting to cross")
		}},
		{"pedestrian_complete", "Pedestrian crossing complete", func() {
			fmt.Println("  ✅ Pedestrian safely crossed")
		}},
	}

	for i, step := range steps {
		fmt.Printf("\n-- Step %d: %s ('%s') --\n", i+1, step.description, step.event)
		if step.setup != nil {
			step.setup()
		}
		res := m.SendEvent(step.event, nil)
		if !res.Success() {
			fmt.Println("rejected:", res.RejectionReason)
		}
		fmt.Printf("Active: %v\n", m.GetActiveStates())
		fmt.Printf("Lights: NS=%s, EW=%s\n", intersection.NorthSouth.Color, intersection.EastWest.Color)
		time.Sleep(200 * time.Millisecond)
	}
}

func runMaintenanceScenario(def fluo.MachineDefinition) {
	m := def.CreateInstance()
	m.AddObserver(&TrafficLightObserver{})

	intersection := &Intersection{
		ID:         "MAIN-001",
		NorthSouth: TrafficLight{ID: "NS-001", Direction: NorthSouth, Color: Red},
		EastWest:   TrafficLight{ID: "EW-001", Direction: EastWest, Color: Green},
	}

	_ = m.Start()
	m.Context().Set("intersection", intersection)

	steps := []struct {
		event       string
		description string
		setup       func()
	}{
		{"power_on", "Power on system", nil},
		{"system_ready", "System ready", nil},
		{"maintenance_request", "Enter maintenance mode", func() {
			fmt.Println("  🔧 Maintenance crew arriving")
		}},
		{"power_off", "System shutdown during maintenance", func() {
			fmt.Println("  🔌 Emergency shutdown requested")
		}},
	}

	for i, step := range steps {
		fmt.Printf("\n-- Step %d: %s ('%s') --\n", i+1, step.description, step.event)
		if step.setup != nil {
			step.setup()
		}
		res := m.SendEvent(step.event, nil)
		if !res.Success() {
			fmt.Println("rejected:", res.RejectionReason)
		}
		fmt.Printf("Active: %v\n", m.GetActiveStates())
		fmt.Printf("Lights: NS=%s, EW=%s\n", intersection.NorthSouth.Color, intersection.EastWest.Color)
		time.Sleep(200 * time.Millisecond)
	}
}
