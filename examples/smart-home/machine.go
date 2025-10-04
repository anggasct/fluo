package main

import (
	"github.com/anggasct/fluo"
)

func BuildSmartHomeMachine() fluo.MachineDefinition {
	b := fluo.NewMachine()

	b.State("off").Initial().
		OnEntry(log("Smart home system powered off")).
		To("startup").On("power_on")

	b.State("startup").
		OnEntry(log("Smart home system starting up...")).
		To("operational").On("system_ready").Do(initializeSystem)

	op := b.ParallelState("operational")

	sec := op.Region("security")
	sec.State("disarmed").Initial().
		OnEntry(log("Security system disarmed"))

	sec.State("disarmed").
		To("operational.security.armed_stay").On("arm_stay").Do(armStayMode).
		To("operational.security.armed_away").On("arm_away").Do(armAwayMode)

	sec.State("armed_stay").
		OnEntry(log("Security armed - stay mode")).
		To("operational.security.disarmed").On("disarm").Do(disarmSystem).
		To("emergency").On("security_breach").When(isSecurityBreach)

	sec.State("armed_away").
		OnEntry(log("Security armed - away mode")).
		To("operational.security.disarmed").On("disarm").Do(disarmSystem).
		To("emergency").On("security_breach").When(isSecurityBreach)

	// Let's try a different approach - create the transition at the parallel state level
	op.To("emergency").On("motion_detected").When(isArmedAway)

	climate := op.Region("climate")
	climate.State("off").Initial().
		OnEntry(log("Climate control off")).
		To("operational.climate.auto").On("climate_on").Do(enableClimate)

	climate.State("heating").
		OnEntry(log("Heating system active")).
		To("operational.climate.off").On("climate_off").
		To("operational.climate.cooling").On("temp_high").When(isTempHigh).
		To("operational.climate.auto").On("auto_mode")

	climate.State("cooling").
		OnEntry(log("Cooling system active")).
		To("operational.climate.off").On("climate_off").
		To("operational.climate.heating").On("temp_low").When(isTempLow).
		To("operational.climate.auto").On("auto_mode")

	climate.State("auto").
		OnEntry(log("Climate control in auto mode")).
		To("operational.climate.heating").On("temp_low").When(isTempLow).
		To("operational.climate.cooling").On("temp_high").When(isTempHigh).
		To("operational.climate.off").On("climate_off")

	lighting := op.Region("lighting")
	lighting.State("manual").Initial().
		OnEntry(log("Lighting in manual mode")).
		To("operational.lighting.schedule").On("schedule_mode").Do(enableScheduledLighting).
		To("operational.lighting.motion_detect").On("motion_mode").Do(enableMotionLighting)

	lighting.State("schedule").
		OnEntry(log("Scheduled lighting active")).
		To("operational.lighting.manual").On("manual_mode").
		To("operational.lighting.motion_detect").On("motion_mode").Do(enableMotionLighting)

	lighting.State("motion_detect").
		OnEntry(log("Motion-based lighting active")).
		To("operational.lighting.manual").On("manual_mode").
		To("operational.lighting.schedule").On("schedule_mode").Do(enableScheduledLighting)

	op.End()

	b.ParallelState("operational").
		To("maintenance").On("maintenance_mode").
		To("emergency").On("emergency_alert").
		To("off").On("power_off").Do(shutdownSystem)

	b.ParallelState("operational").
		To("operational").On("temp_high").
		To("operational").On("temp_low").
		To("operational").On("emergency_clear").
		To("operational").On("motion_mode")

	b.State("maintenance").
		OnEntry(log("System in maintenance mode")).
		To("operational").On("maintenance_complete").
		To("off").On("power_off")

	b.State("emergency").
		OnEntry(log("EMERGENCY: System in alert mode")).
		To("operational").On("emergency_clear").When(isEmergencyClear).Do(activateEmergencyProtocols).
		To("off").On("force_shutdown")

	return b.Build()
}
