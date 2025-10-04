package main

import "time"

type LightColor string

const (
	Red    LightColor = "red"
	Yellow LightColor = "yellow"
	Green  LightColor = "green"
)

type Direction string

const (
	NorthSouth Direction = "north_south"
	EastWest   Direction = "east_west"
)

type TrafficLight struct {
	ID        string
	Direction Direction
	Color     LightColor
	Duration  time.Duration
}

type Intersection struct {
	ID              string
	NorthSouth      TrafficLight
	EastWest        TrafficLight
	PedestrianMode  bool
	EmergencyMode   bool
	MaintenanceMode bool
}

type TimerEvent struct {
	Name     string
	Duration time.Duration
}
