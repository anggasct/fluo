package main

import (
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

func log(message string) fluo.ActionFunc {
	return func(ctx fluo.Context) error {
		fmt.Printf("[LOG] %s\n", message)
		return nil
	}
}

func getIntersection(ctx fluo.Context) *Intersection {
	if v, ok := ctx.Get("intersection"); ok {
		if i, ok := v.(*Intersection); ok {
			return i
		}
	}
	return nil
}

func initializeIntersection(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.NorthSouth.Color = Red
		intersection.EastWest.Color = Green
		intersection.EmergencyMode = false
		intersection.PedestrianMode = false
		intersection.MaintenanceMode = false
	}

	ctx.Set("ns_timer", time.Now().Add(30*time.Second))
	ctx.Set("ew_timer", time.Now().Add(25*time.Second))

	fmt.Println("Intersection initialized - NS:RED, EW:GREEN")
	return nil
}

func shutdownSystem(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.NorthSouth.Color = Red
		intersection.EastWest.Color = Red
	}
	fmt.Println("System shutdown - all lights RED")
	return nil
}

func setNorthSouthRedYellow(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.NorthSouth.Color = Yellow
	}
	ctx.Set("ns_timer", time.Now().Add(3*time.Second))
	fmt.Println("North-South: RED+YELLOW (3s)")
	return nil
}

func setNorthSouthGreen(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.NorthSouth.Color = Green
	}
	ctx.Set("ns_timer", time.Now().Add(25*time.Second))
	fmt.Println("North-South: GREEN (25s)")
	return nil
}

func setNorthSouthYellow(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.NorthSouth.Color = Yellow
	}
	ctx.Set("ns_timer", time.Now().Add(5*time.Second))
	fmt.Println("North-South: YELLOW (5s)")
	return nil
}

func setNorthSouthRed(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.NorthSouth.Color = Red
	}
	ctx.Set("ns_timer", time.Now().Add(30*time.Second))
	fmt.Println("North-South: RED (30s)")
	return nil
}

func setEastWestGreen(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.EastWest.Color = Green
	}
	ctx.Set("ew_timer", time.Now().Add(25*time.Second))
	fmt.Println("East-West: GREEN (25s)")
	return nil
}

func setEastWestYellow(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.EastWest.Color = Yellow
	}
	ctx.Set("ew_timer", time.Now().Add(5*time.Second))
	fmt.Println("East-West: YELLOW (5s)")
	return nil
}

func setEastWestRed(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.EastWest.Color = Red
	}
	ctx.Set("ew_timer", time.Now().Add(30*time.Second))
	fmt.Println("East-West: RED (30s)")
	return nil
}

func setEastWestRedYellow(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.EastWest.Color = Yellow
	}
	ctx.Set("ew_timer", time.Now().Add(3*time.Second))
	fmt.Println("East-West: RED+YELLOW (3s)")
	return nil
}

func activateEmergencyMode(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.EmergencyMode = true
		intersection.NorthSouth.Color = Red
		intersection.EastWest.Color = Red
	}
	ctx.Set("emergency_active", true)
	fmt.Println("Emergency mode activated - all RED")
	return nil
}

func activatePedestrianMode(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.PedestrianMode = true
	}
	ctx.Set("pedestrian_timer", time.Now().Add(15*time.Second))
	fmt.Println("Pedestrian mode activated - extended crossing")
	return nil
}

func activateMaintenanceMode(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.MaintenanceMode = true
		intersection.NorthSouth.Color = Yellow
		intersection.EastWest.Color = Yellow
	}
	fmt.Println("Maintenance mode activated - flashing yellow")
	return nil
}

func resumeNormalOperation(ctx fluo.Context) error {
	intersection := getIntersection(ctx)
	if intersection != nil {
		intersection.EmergencyMode = false
		intersection.PedestrianMode = false
		intersection.MaintenanceMode = false
	}
	ctx.Set("emergency_active", false)
	fmt.Println("Resuming normal operation")
	return nil
}

func isNorthSouthRedExpired(ctx fluo.Context) bool {
	if timer, ok := ctx.Get("ns_timer"); ok {
		if t, ok := timer.(time.Time); ok {
			return time.Now().After(t)
		}
	}
	return false
}

func isNorthSouthRedYellowExpired(ctx fluo.Context) bool {
	if timer, ok := ctx.Get("ns_timer"); ok {
		if t, ok := timer.(time.Time); ok {
			return time.Now().After(t)
		}
	}
	return false
}

func isNorthSouthGreenExpired(ctx fluo.Context) bool {
	if timer, ok := ctx.Get("ns_timer"); ok {
		if t, ok := timer.(time.Time); ok {
			return time.Now().After(t)
		}
	}
	return false
}

func isNorthSouthYellowExpired(ctx fluo.Context) bool {
	if timer, ok := ctx.Get("ns_timer"); ok {
		if t, ok := timer.(time.Time); ok {
			return time.Now().After(t)
		}
	}
	return false
}

func isEastWestGreenExpired(ctx fluo.Context) bool {
	if timer, ok := ctx.Get("ew_timer"); ok {
		if t, ok := timer.(time.Time); ok {
			return time.Now().After(t)
		}
	}
	return false
}

func isEastWestYellowExpired(ctx fluo.Context) bool {
	if timer, ok := ctx.Get("ew_timer"); ok {
		if t, ok := timer.(time.Time); ok {
			return time.Now().After(t)
		}
	}
	return false
}

func isEastWestRedExpired(ctx fluo.Context) bool {
	if timer, ok := ctx.Get("ew_timer"); ok {
		if t, ok := timer.(time.Time); ok {
			return time.Now().After(t)
		}
	}
	return false
}

func isEastWestRedYellowExpired(ctx fluo.Context) bool {
	if timer, ok := ctx.Get("ew_timer"); ok {
		if t, ok := timer.(time.Time); ok {
			return time.Now().After(t)
		}
	}
	return false
}

func isEmergencyClear(ctx fluo.Context) bool {
	if v, ok := ctx.Get("emergency_clear"); ok {
		return v.(bool)
	}
	return false
}
