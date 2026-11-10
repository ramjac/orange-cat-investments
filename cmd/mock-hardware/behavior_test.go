package main

import (
	"math"
	"math/rand"
	"testing"
)

func TestBehaviorModel_Transitions(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	b := NewBehaviorModel(rng)

	counts := make(map[string]int)
	iterations := 10000

	for i := 0; i < iterations; i++ {
		counts[b.nextActivity()]++
	}

	// Because sleeping has 0.8 to stay sleeping, it should be the most common
	if counts["sleeping"] < counts["zooming"] {
		t.Errorf("expected sleeping to be more common than zooming")
	}

	// Just checking valid transitions occurred
	total := 0
	for _, act := range b.activities {
		total += counts[act]
	}
	if total != iterations {
		t.Errorf("expected %d transitions, got %d", iterations, total)
	}
}

func TestBehaviorModel_Environment(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	b := NewBehaviorModel(rng)

	for i := 0; i < 1000; i++ {
		b.nextActivity()
		temp, humidity, lux, noise := b.NextEnvironment()

		if temp < 20.0 || temp > 30.0 {
			t.Errorf("temp out of bounds: %v", temp)
		}
		if humidity < 30.0 || humidity > 70.0 {
			t.Errorf("humidity out of bounds: %v", humidity)
		}
		if lux < 100.0 || lux > 400.0 {
			t.Errorf("lux out of bounds: %v", lux)
		}
		if noise < 10.0 || noise > 90.0 {
			t.Errorf("noise out of bounds: %v", noise)
		}
	}
}

func TestBehaviorModel_EnvironmentCorrelates(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	b := NewBehaviorModel(rng)

	// Force state to zooming, step a bunch of times, noise should trend towards 70
	b.currentActivity = "zooming"
	for i := 0; i < 50; i++ {
		b.NextEnvironment()
	}
	if math.Abs(b.noise-70.0) > 10.0 {
		t.Errorf("expected noise near 70.0 when zooming, got %v", b.noise)
	}

	// Force state to sleeping, step a bunch of times, noise should trend towards 20
	b.currentActivity = "sleeping"
	for i := 0; i < 50; i++ {
		b.NextEnvironment()
	}
	if math.Abs(b.noise-20.0) > 10.0 {
		t.Errorf("expected noise near 20.0 when sleeping, got %v", b.noise)
	}
}

func TestBehaviorModel_Envelope(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	b := NewBehaviorModel(rng)

	env := b.NextActivityEnvelope("cam-1", "fel-1")
	if env.EventID == "" {
		t.Errorf("expected valid EventID")
	}
	// uuid V7 check: V7 UUIDs have version 7 at the 13th character (index 14 with hyphens)
	if len(env.EventID) == 36 && env.EventID[14] != '7' {
		t.Errorf("expected UUIDv7 EventID, got %v", env.EventID)
	}

	if env.OccurredAt.IsZero() || env.OccurredAt.Location().String() != "UTC" {
		t.Errorf("expected valid UTC OccurredAt")
	}

	if env.Payload.CameraID != "cam-1" {
		t.Errorf("expected cameraID cam-1, got %v", env.Payload.CameraID)
	}
	if env.Payload.FelineID != "fel-1" {
		t.Errorf("expected felineID fel-1, got %v", env.Payload.FelineID)
	}
	if env.Payload.ActivityType == "" {
		t.Errorf("expected non-empty ActivityType")
	}
	if env.Payload.ConfidenceScore < 0.8 || env.Payload.ConfidenceScore >= 1.0 {
		t.Errorf("expected confidence between 0.8 and 1.0, got %v", env.Payload.ConfidenceScore)
	}
}
