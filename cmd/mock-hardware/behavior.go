package main

import (
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/orange-cat-investments/oci/pkg/envelope"
)

type Transition struct {
	State       string
	Probability float64
}

type BehaviorModel struct {
	mu  sync.Mutex
	rng *rand.Rand

	currentActivity string
	activities      []string
	matrix          map[string][]Transition

	tempC    float64
	humidity float64
	lux      float64
	noise    float64
}

func NewBehaviorModel(rng *rand.Rand) *BehaviorModel {
	activities := []string{"zooming", "sleeping", "eating", "playing", "grooming"}

	// Use an ordered slice of transitions to ensure deterministic behavior when seeded
	matrix := map[string][]Transition{
		"sleeping": {
			{"sleeping", 0.8},
			{"grooming", 0.1},
			{"eating", 0.05},
			{"playing", 0.03},
			{"zooming", 0.02},
		},
		"grooming": {
			{"sleeping", 0.3},
			{"grooming", 0.5},
			{"eating", 0.1},
			{"playing", 0.05},
			{"zooming", 0.05},
		},
		"eating": {
			{"sleeping", 0.4},
			{"grooming", 0.3},
			{"eating", 0.1},
			{"playing", 0.1},
			{"zooming", 0.1},
		},
		"playing": {
			{"sleeping", 0.2},
			{"grooming", 0.1},
			{"eating", 0.2},
			{"playing", 0.4},
			{"zooming", 0.1},
		},
		"zooming": {
			{"sleeping", 0.4},
			{"grooming", 0.1},
			{"eating", 0.2},
			{"playing", 0.2},
			{"zooming", 0.1},
		},
	}

	return &BehaviorModel{
		rng:             rng,
		currentActivity: activities[rng.Intn(len(activities))],
		activities:      activities,
		matrix:          matrix,
		tempC:           22.0 + rng.Float64()*3.0,
		humidity:        40.0 + rng.Float64()*20.0,
		lux:             200.0 + rng.Float64()*100.0,
		noise:           30.0 + rng.Float64()*20.0,
	}
}

func (b *BehaviorModel) nextActivity() string {
	r := b.rng.Float64()
	var cumulative float64
	for _, transition := range b.matrix[b.currentActivity] {
		cumulative += transition.Probability
		if r <= cumulative {
			b.currentActivity = transition.State
			return transition.State
		}
	}
	b.currentActivity = b.activities[b.rng.Intn(len(b.activities))]
	return b.currentActivity
}

func (b *BehaviorModel) NextActivityEnvelope(cameraID, felineID string) envelope.EventEnvelope[envelope.CatSpottedPayload] {
	b.mu.Lock()
	defer b.mu.Unlock()

	activity := b.nextActivity()
	confidence := 0.8 + b.rng.Float64()*0.19 // 0.80 to 0.99

	return envelope.EventEnvelope[envelope.CatSpottedPayload]{
		EventID:       uuid.Must(uuid.NewV7()).String(),
		EventType:     "observation.cat_spotted.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "correlation-trace-" + uuid.Must(uuid.NewV7()).String()[:8],
		Payload: envelope.CatSpottedPayload{
			CameraID:        cameraID,
			FelineID:        felineID,
			ActivityType:    activity,
			ConfidenceScore: confidence,
		},
	}
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func (b *BehaviorModel) NextEnvironment() (tempC, humidityPct, lux, noiseDB float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Drift
	b.tempC += (b.rng.Float64()*2.0 - 1.0) * 0.5       // +/- 0.5
	b.humidity += (b.rng.Float64()*2.0 - 1.0) * 2.0 // +/- 2.0
	b.lux += (b.rng.Float64()*2.0 - 1.0) * 10.0     // +/- 10.0
	b.noise += (b.rng.Float64()*2.0 - 1.0) * 2.0    // +/- 2.0

	// Bounds
	b.tempC = clamp(b.tempC, 20.0, 30.0)
	b.humidity = clamp(b.humidity, 30.0, 70.0)
	b.lux = clamp(b.lux, 100.0, 400.0)

	// Correlate noise with activity
	targetNoise := 30.0
	switch b.currentActivity {
	case "sleeping":
		targetNoise = 20.0
	case "eating", "grooming":
		targetNoise = 35.0
	case "playing":
		targetNoise = 50.0
	case "zooming":
		targetNoise = 70.0
	}

	// move current noise towards target
	b.noise = b.noise + (targetNoise - b.noise) * 0.2
	b.noise = clamp(b.noise, 10.0, 90.0)

	return b.tempC, b.humidity, b.lux, b.noise
}
