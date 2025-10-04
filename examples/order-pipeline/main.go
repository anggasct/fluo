package main

import (
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

type OrderObserver struct{ fluo.BaseObserver }

func (o *OrderObserver) OnTransition(from, to string, event fluo.Event, ctx fluo.Context) {
	name := ""
	if event != nil {
		name = event.GetName()
	}
	fmt.Printf("%s --[%s]--> %s\n", from, name, to)
}

func (o *OrderObserver) OnStateEnter(state string, ctx fluo.Context) {
	fmt.Printf("Entered: %s\n", state)
}

func main() {
	def := BuildOrderMachine()
	m := def.CreateInstance()
	m.AddObserver(&OrderObserver{})

	fmt.Println("=== Scenario 1: Digital Order ===")
	m.Context().Set("order", &Order{ID: "ORD-1001", ItemType: Digital, Amount: 3999, InStock: true})
	_ = m.Start()
	runDigitalFlow(m)

	fmt.Println("\n=== Scenario 2: Physical Order ===")
	_ = m.Reset()
	_ = m.Start()
	m.Context().Set("order", &Order{ID: "ORD-1002", ItemType: Physical, Amount: 12999, InStock: true})
	runPhysicalFlow(m)
}

func runDigitalFlow(m fluo.Machine) {
	steps := []string{
		"place",
		"pay_ok",
		"risk_ok",
		"deliver",
	}
	for i, ev := range steps {
		fmt.Printf("\n-- Step %d: send '%s' --\n", i+1, ev)
		res := m.SendEvent(ev, nil)
		if !res.Success() {
			fmt.Println("rejected:", res.RejectionReason)
		}
		fmt.Println("Active:", m.GetActiveStates())
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Println("Current state:", m.CurrentState())
}

func runPhysicalFlow(m fluo.Machine) {
	steps := []string{
		"place",
		"pay_ok",
		"risk_ok",
		"pack_done",
		"label_done",
		"ship",
	}
	for i, ev := range steps {
		fmt.Printf("\n-- Step %d: send '%s' --\n", i+1, ev)
		res := m.SendEvent(ev, nil)
		if !res.Success() {
			fmt.Println("rejected:", res.RejectionReason)
		}
		fmt.Println("Active:", m.GetActiveStates())
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Println("Current state:", m.CurrentState())
}
