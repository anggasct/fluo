package main

import (
	"github.com/anggasct/fluo"
)

func BuildTrafficLightMachine() fluo.MachineDefinition {
	b := fluo.NewMachine()

	b.State("off").Initial().
		OnEntry(log("Traffic light system OFF")).
		To("startup").On("power_on")

	b.State("startup").
		OnEntry(log("Traffic light system starting up...")).
		To("normal_operation").On("system_ready").Do(initializeIntersection)

	op := b.ParallelState("normal_operation")

	ns := op.Region("north_south")
	ns.State("red").Initial().
		OnEntry(log("North-South: RED light")).
		To("red_yellow").On("timer_expired").When(isNorthSouthRedExpired).Do(setNorthSouthRedYellow)

	ns.State("red_yellow").
		OnEntry(log("North-South: RED+YELLOW light")).
		To("green").On("timer_expired").When(isNorthSouthRedYellowExpired).Do(setNorthSouthGreen)

	ns.State("green").
		OnEntry(log("North-South: GREEN light")).
		To("yellow").On("timer_expired").When(isNorthSouthGreenExpired).Do(setNorthSouthYellow)

	ns.State("yellow").
		OnEntry(log("North-South: YELLOW light")).
		To("red").On("timer_expired").When(isNorthSouthYellowExpired).Do(setNorthSouthRed)

	ew := op.Region("east_west")
	ew.State("green").Initial().
		OnEntry(log("East-West: GREEN light")).
		To("yellow").On("timer_expired").When(isEastWestGreenExpired).Do(setEastWestYellow)

	ew.State("yellow").
		OnEntry(log("East-West: YELLOW light")).
		To("red").On("timer_expired").When(isEastWestYellowExpired).Do(setEastWestRed)

	ew.State("red").
		OnEntry(log("East-West: RED light")).
		To("red_yellow").On("timer_expired").When(isEastWestRedExpired).Do(setEastWestRedYellow)

	ew.State("red_yellow").
		OnEntry(log("East-West: RED+YELLOW light")).
		To("green").On("timer_expired").When(isEastWestRedYellowExpired).Do(setEastWestGreen)

	op.
		To("emergency_mode").On("emergency_vehicle").Do(activateEmergencyMode).
		To("pedestrian_mode").On("pedestrian_button").Do(activatePedestrianMode).
		To("maintenance_mode").On("maintenance_request").Do(activateMaintenanceMode).
		To("off").On("power_off").Do(shutdownSystem)

	op.End()

	b.State("emergency_mode").
		OnEntry(log("EMERGENCY MODE: All directions RED")).
		To("normal_operation").On("emergency_clear").When(isEmergencyClear).Do(resumeNormalOperation)

	b.State("pedestrian_mode").
		OnEntry(log("PEDESTRIAN MODE: Extended crossing time")).
		To("normal_operation").On("pedestrian_complete").Do(resumeNormalOperation)

	b.State("maintenance_mode").
		OnEntry(log("MAINTENANCE MODE: Flashing yellow")).
		To("normal_operation").On("maintenance_complete").Do(resumeNormalOperation)

	b.State("maintenance_mode").
		To("off").On("power_off").Do(shutdownSystem)

	return b.Build()
}
