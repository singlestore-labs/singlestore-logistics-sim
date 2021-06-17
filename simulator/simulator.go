package simulator

import (
	"log"
	"math"
	"math/rand"
	"simulator/enum"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	uuid "github.com/satori/go.uuid"
	"gonum.org/v1/gonum/stat/distuv"
)

const (
	VerboseSilent = 0
	VerboseInfo   = 1
	VerboseDebug  = 2
	VerboseSilly  = 3
)

type State struct {
	Clock     *Clock
	Trackers  []Tracker
	Locations *LocationIndex
	Topics    *Topics

	// CloseCh should be closed to stop the Simulation
	CloseCh chan struct{}

	SimulatorID string
	SimInterval time.Duration
	Verbose     int

	MaxPackages             int
	MaxDelivered            int
	PackagesPerTick         *distuv.Normal
	HoursAtRest             *distuv.Normal
	ProbabilityExpress      float64
	MinShippingDistanceKM   float64
	MinAirFreightDistanceKM float64
	AvgLandSpeedKMPH        float64
	AvgAirSpeedKMPH         float64
}

func NewState(c *Config, locations *LocationIndex, producer Producer, initialTrackers []Tracker) *State {
	return &State{
		Clock:     NewClock(c.StartTime),
		Trackers:  initialTrackers,
		Locations: locations,
		Topics:    NewTopics(producer),

		CloseCh: make(chan struct{}),

		SimulatorID: c.SimulatorID,
		SimInterval: c.SimInterval,
		Verbose:     c.Verbose,

		MaxPackages:             c.MaxPackages,
		MaxDelivered:            c.MaxDelivered,
		PackagesPerTick:         c.PackagesPerTick.ToDist(),
		HoursAtRest:             c.HoursAtRest.ToDist(),
		ProbabilityExpress:      c.ProbabilityExpress,
		MinShippingDistanceKM:   c.MinShippingDistanceKM,
		MinAirFreightDistanceKM: c.MinAirFreightDistanceKM,
		AvgLandSpeedKMPH:        c.AvgLandSpeedKMPH,
		AvgAirSpeedKMPH:         c.AvgAirSpeedKMPH,
	}
}

func Simulate(state *State) {
	totalDelivered := 0
	for {
		now := state.Clock.Now()

		if state.Verbose >= VerboseInfo {
			log.Printf("TICK: %s tracked(%d) delivered(%d/%d)", now, len(state.Trackers), totalDelivered, state.MaxDelivered)
		}

		if state.MaxPackages <= 0 || len(state.Trackers) < state.MaxPackages {
			numNewPackages := state.PackagesPerTick.Rand()
			if state.MaxPackages > 0 {
				numNewPackages = math.Min(
					float64(state.MaxPackages-len(state.Trackers)),
					numNewPackages,
				)
			}
			CreatePackages(state, now, int(numNewPackages))
		}

		// keep track of the number of delivered packages; the corresponding
		// trackers will need to be removed before our next Tick
		numDelivered := 0

		// transition packages
		for i := range state.Trackers {
			// for performance we mutate the tracker in place
			tracker := &state.Trackers[i]

			if now.Equal(tracker.NextTransitionTime) || now.After(tracker.NextTransitionTime) {
				switch tracker.State {
				case enum.AtRest:
					TriggerDepartureScan(state, tracker)

				case enum.InTransit:
					// the package has reached it's current destination
					if tracker.DestinationLocationID == tracker.NextLocationID {
						// the package has reached it's final destination
						TriggerDelivered(state, tracker)
						numDelivered++
					} else {
						// the package has reached a interim destination
						TriggerArrivalScan(state, tracker)
					}

				default:
					log.Panicf("unknown state %+v for package %s", tracker.State, tracker.PackageID)
				}
			}
		}

		// remove trackers for delivered packages and determine the next transition time
		nextTransitionTime := state.Trackers[0].NextTransitionTime
		newTrackers := make([]Tracker, 0, len(state.Trackers)-numDelivered)
		for i := range state.Trackers {
			tracker := &state.Trackers[i]
			if tracker.NextTransitionTime.Before(nextTransitionTime) {
				nextTransitionTime = tracker.NextTransitionTime
			}
			if !tracker.Delivered {
				newTrackers = append(newTrackers, *tracker)
			}
		}
		state.Trackers = newTrackers

		// advance the clock
		state.Clock.Set(nextTransitionTime)

		totalDelivered += numDelivered

		if state.MaxDelivered > 0 && totalDelivered >= state.MaxDelivered {
			return
		}

		select {
		case <-state.CloseCh:
			return
		default:
		}

		if state.SimInterval > 0 {
			time.Sleep(state.SimInterval)
		}
	}
}

func CreatePackages(state *State, now time.Time, numNewPackages int) {
	// create new packages
	for i := 0; i < numNewPackages; i++ {
		method := enum.Standard
		if rand.Float64() > state.ProbabilityExpress {
			method = enum.Express
		}

		origin := state.Locations.Rand(nil)

		destination := state.Locations.Rand(func(p orb.Pointer) bool {
			candidate := p.(*Location)
			if candidate == origin {
				return false
			}
			// we only deliver packages which travel farther than MinShippingDistanceKM
			return geo.Distance(origin.Position, candidate.Position)/1000 > state.MinShippingDistanceKM
		})

		// extremely crude delivery estimate calculation
		distance := geo.Distance(origin.Position, destination.Position) / 1000
		avgSpeed := state.AvgLandSpeedKMPH
		if method == enum.Express {
			avgSpeed = state.AvgAirSpeedKMPH
		}
		// include an overhead buffer of 20% due to processing delays per transit point
		hours := (distance / avgSpeed) * 1.2
		deliveryEstimate := now.Add(time.Duration(hours) * time.Hour)

		pkg := Package{
			PackageID:             uuid.NewV4(),
			SimulatorID:           state.SimulatorID,
			Received:              now,
			OriginLocationID:      origin.LocationID,
			DestinationLocationID: destination.LocationID,
			DeliveryEstimate:      deliveryEstimate,
			Method:                method,
		}

		err := state.Topics.WritePackage(&pkg)
		if err != nil {
			log.Panicf("failed to write package to topic: %v", err)
		}

		nextTransitionTime := now.Add(time.Hour * time.Duration(state.HoursAtRest.Rand()))

		state.Trackers = append(state.Trackers, Tracker{
			PackageID:             pkg.PackageID,
			Method:                pkg.Method,
			DestinationLocationID: pkg.DestinationLocationID,

			State:          enum.InTransit,
			Seq:            0,
			LastLocationID: pkg.OriginLocationID,
			NextLocationID: pkg.OriginLocationID,

			NextTransitionTime: nextTransitionTime,
		})

		if state.Verbose >= VerboseDebug {
			log.Printf("CreatePackage(%s): %s -> %s (%s, %.1fkm)",
				pkg.PackageID.String()[:8],
				PointString(origin.Position),
				PointString(destination.Position),
				method, distance)
		}

		TriggerArrivalScan(state, &state.Trackers[len(state.Trackers)-1])
	}

}

func TriggerDepartureScan(state *State, t *Tracker) {
	if t.State != enum.AtRest {
		log.Panicf("TriggerDepartureScan can only be called when State == AtRest")
	}

	currentLocation, err := state.Locations.Lookup(t.LastLocationID)
	if err != nil {
		log.Panic(err)
	}
	destinationLocation, err := state.Locations.Lookup(t.DestinationLocationID)
	if err != nil {
		log.Panic(err)
	}

	nextLocation := state.Locations.NextLocation(currentLocation, destinationLocation, t.Method)

	distanceToNext := geo.Distance(currentLocation.Position, nextLocation.Position) / 1000
	speed := state.AvgLandSpeedKMPH
	if distanceToNext > state.MinAirFreightDistanceKM {
		speed = state.AvgAirSpeedKMPH
	}

	duration := time.Hour * time.Duration(distanceToNext/speed)
	nextTransitionTime := state.Clock.Now().Add(duration)

	// update tracker state fields
	t.State = enum.InTransit
	t.Seq = t.Seq + 1
	t.LastLocationID = currentLocation.LocationID

	// update tracker InTransit fields
	t.NextTransitionTime = nextTransitionTime
	t.NextLocationID = nextLocation.LocationID

	if state.Verbose >= VerboseDebug {
		log.Printf("DepartureScan(%s): %s -> %s in %s (%.1fkm)",
			t.PackageID.String()[:8],
			PointString(currentLocation.Position),
			PointString(nextLocation.Position),
			t.NextTransitionTime.Sub(state.Clock.Now()),
			distanceToNext)
	}

	err = state.Topics.WriteTransition(state.Clock.Now(), enum.DepartureScan, t)
	if err != nil {
		log.Panicf("failed to write transition to topic: %v", err)
	}
}

func TriggerArrivalScan(state *State, t *Tracker) {
	if t.State != enum.InTransit {
		log.Panicf("TriggerArrivalScan can only be called when State == InTransit")
	}

	// update tracker state fields
	t.State = enum.AtRest
	t.Seq = t.Seq + 1
	t.LastLocationID = t.NextLocationID

	// update tracker AtRest fields
	now := state.Clock.Now()
	t.NextTransitionTime = now.Add(time.Hour * time.Duration(state.HoursAtRest.Rand()))

	if state.Verbose >= VerboseDebug {
		currentLocation, err := state.Locations.Lookup(t.LastLocationID)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("ArrivalScan(%s): %s; departure in %s",
			t.PackageID.String()[:8],
			PointString(currentLocation.Position),
			t.NextTransitionTime.Sub(now))
	}

	err := state.Topics.WriteTransition(now, enum.ArrivalScan, t)
	if err != nil {
		log.Panicf("failed to write transition to topic: %v", err)
	}
}

func TriggerDelivered(state *State, t *Tracker) {
	if t.State != enum.InTransit {
		log.Panicf("TriggerDelivered can only be called when State == InTransit")
	}

	// update tracker state fields
	t.Delivered = true
	t.State = enum.AtRest
	t.Seq = t.Seq + 1
	t.LastLocationID = t.NextLocationID

	if state.Verbose >= VerboseDebug {
		currentLocation, err := state.Locations.Lookup(t.LastLocationID)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("Delivered(%s): %s",
			t.PackageID.String()[:8],
			PointString(currentLocation.Position))
	}

	err := state.Topics.WriteTransition(state.Clock.Now(), enum.Delivered, t)
	if err != nil {
		log.Panicf("failed to write transition to topic: %v", err)
	}
}
