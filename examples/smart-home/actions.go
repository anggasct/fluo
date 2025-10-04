package main

import (
	"fmt"

	"github.com/anggasct/fluo"
)

func log(message string) fluo.ActionFunc {
	return func(ctx fluo.Context) error {
		fmt.Printf("[LOG] %s\n", message)
		return nil
	}
}

func getSmartHome(ctx fluo.Context) *SmartHome {
	if v, ok := ctx.Get("smart_home"); ok {
		if sh, ok := v.(*SmartHome); ok {
			return sh
		}
	}
	return nil
}

func getSensorData(ctx fluo.Context) *SensorData {
	if v, ok := ctx.Get("sensor_data"); ok {
		if sd, ok := v.(*SensorData); ok {
			return sd
		}
	}
	return nil
}

func initializeSystem(ctx fluo.Context) error {
	sh := getSmartHome(ctx)
	if sh != nil {
		sh.SecurityMode = Disarmed
		sh.ClimateMode = ClimateOff
		sh.LightingMode = Manual
		sh.EmergencyActive = false
	}
	fmt.Println("System initialized")
	return nil
}

func shutdownSystem(ctx fluo.Context) error {
	sh := getSmartHome(ctx)
	if sh != nil {
		sh.SecurityMode = Disarmed
		sh.ClimateMode = ClimateOff
		sh.EmergencyActive = false
	}
	fmt.Println("System shutdown")
	return nil
}

func armStayMode(ctx fluo.Context) error {
	sh := getSmartHome(ctx)
	if sh != nil {
		sh.SecurityMode = ArmedStay
	}
	ctx.Set("security_armed", true)
	fmt.Println("Security armed in stay mode")
	return nil
}

func armAwayMode(ctx fluo.Context) error {
	sh := getSmartHome(ctx)
	if sh != nil {
		sh.SecurityMode = ArmedAway
	}
	ctx.Set("security_armed", true)
	fmt.Println("Security armed in away mode")
	return nil
}

func disarmSystem(ctx fluo.Context) error {
	sh := getSmartHome(ctx)
	if sh != nil {
		sh.SecurityMode = Disarmed
	}
	ctx.Set("security_armed", false)
	fmt.Println("Security system disarmed")
	return nil
}

func enableClimate(ctx fluo.Context) error {
	sh := getSmartHome(ctx)
	if sh != nil {
		sh.ClimateMode = Auto
	}
	ctx.Set("climate_active", true)
	fmt.Println("Climate control enabled")
	return nil
}

func enableScheduledLighting(ctx fluo.Context) error {
	sh := getSmartHome(ctx)
	if sh != nil {
		sh.LightingMode = Schedule
	}
	ctx.Set("schedule_lighting", true)
	fmt.Println("Scheduled lighting enabled")
	return nil
}

func enableMotionLighting(ctx fluo.Context) error {
	sh := getSmartHome(ctx)
	if sh != nil {
		sh.LightingMode = MotionDetect
	}
	ctx.Set("motion_lighting", true)
	fmt.Println("Motion-based lighting enabled")
	return nil
}

func activateEmergencyProtocols(ctx fluo.Context) error {
	sh := getSmartHome(ctx)
	if sh != nil {
		sh.EmergencyActive = true
	}
	ctx.Set("emergency_active", true)
	fmt.Println("Emergency protocols activated")
	return nil
}

func isSecurityBreach(ctx fluo.Context) bool {
	sd := getSensorData(ctx)
	sh := getSmartHome(ctx)
	if sd != nil && sh != nil {
		return (sh.SecurityMode == ArmedStay && sd.DoorOpen) ||
			(sh.SecurityMode == ArmedAway && (sd.MotionDetected || sd.DoorOpen))
	}
	return false
}

func isArmedAway(ctx fluo.Context) bool {
	sh := getSmartHome(ctx)
	return sh != nil && sh.SecurityMode == ArmedAway
}

func isTempHigh(ctx fluo.Context) bool {
	sd := getSensorData(ctx)
	return sd != nil && sd.Temperature > 75
}

func isTempLow(ctx fluo.Context) bool {
	sd := getSensorData(ctx)
	return sd != nil && sd.Temperature < 68
}

func isEmergencyClear(ctx fluo.Context) bool {
	if v, ok := ctx.Get("emergency_clear"); ok {
		return v.(bool)
	}
	return false
}
