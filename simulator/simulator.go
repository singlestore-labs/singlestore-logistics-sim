package simulator

import (
	"log"
	"math"
	"math/rand"
	"simulator/enum"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
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
	Topics    Topics

	// CloseCh should be closed to stop the Simulation
	CloseCh chan struct{}

	SimulatorID string
	SimInterval time.Duration
	Verbose     int

	MaxPackages                      int
	MaxDelivered                     int
	PackagesPerTick                  *distuv.Normal
	HoursAtRest                      *distuv.Normal
	ProbabilityExpress               float64
	MinShippingDistanceKM            float64
	MinShippingDistanceMetresSquared float64
	MinAirFreightDistanceKM          float64
	AvgLandSpeedKMPH                 float64
	AvgAirSpeedKMPH                  float64
}

func NewState(c *Config, locations *LocationIndex, topics Topics, initialTrackers []Tracker) *State {
	return &State{
		Clock:     NewClock(c.StartTime, c.TickDuration),
		Trackers:  initialTrackers,
		Locations: locations,
		Topics:    topics,

		CloseCh: make(chan struct{}),

		SimulatorID: c.SimulatorID,
		SimInterval: c.SimInterval,
		Verbose:     c.Verbose,

		MaxPackages:                      c.MaxPackages,
		MaxDelivered:                     c.MaxDelivered,
		PackagesPerTick:                  c.PackagesPerTick.ToDist(),
		HoursAtRest:                      c.HoursAtRest.ToDist(),
		ProbabilityExpress:               c.ProbabilityExpress,
		MinShippingDistanceKM:            c.MinShippingDistanceKM,
		MinShippingDistanceMetresSquared: c.MinShippingDistanceKM * 1000 * c.MinShippingDistanceKM * 1000,
		MinAirFreightDistanceKM:          c.MinAirFreightDistanceKM,
		AvgLandSpeedKMPH:                 c.AvgLandSpeedKMPH,
		AvgAirSpeedKMPH:                  c.AvgAirSpeedKMPH,
	}
}

func Simulate(state *State) {
	totalDelivered := 0
	for {
		delta := state.Clock.Tick()
		now := state.Clock.Now()

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

			switch tracker.State {
			case enum.AtRest:
				if now.Equal(tracker.NextTransitionTime) || now.After(tracker.NextTransitionTime) {
					TriggerDepartureScan(state, tracker)
				}

			case enum.InTransit:
				if UpdatePosition(state, tracker, delta) {
					// the package has reached it's current destination
					if tracker.DestinationLocationID == tracker.NextLocationID {
						// the package has reached it's final destination
						TriggerDelivered(state, tracker)
						numDelivered++
					} else {
						// the package has reached a interim destination
						TriggerArrivalScan(state, tracker)
					}
				}
			default:
				log.Panicf("unknown state %+v for package %s", tracker.State, tracker.PackageID)
			}
		}

		// remove trackers for delivered packages
		newTrackers := make([]Tracker, 0, len(state.Trackers)-numDelivered)
		for i := range state.Trackers {
			tracker := &state.Trackers[i]
			if !tracker.Delivered {
				newTrackers = append(newTrackers, *tracker)
			}
		}
		state.Trackers = newTrackers

		totalDelivered += numDelivered

		if state.Verbose >= VerboseDebug {
			log.Printf("TICK: tracked(%d) delivered(%d/%d)", len(state.Trackers), totalDelivered, state.MaxDelivered)
		}

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

		origin, err := state.Locations.Rand(nil)
		if err != nil {
			log.Panic(err)
		}

		destination, err := state.Locations.Rand(func(p orb.Pointer) bool {
			candidate := p.(*Location)
			if candidate == origin {
				return false
			}
			// we only deliver packages which travel farther than MinShippingDistanceKM
			return planar.DistanceSquared(origin.Position, candidate.Position) > state.MinShippingDistanceMetresSquared
		})
		if err != nil {
			log.Panic(err)
		}

		// extremely crude delivery estimate calculation
		distance := planar.Distance(origin.Position, destination.Position) / 1000
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

		err = state.Topics.WritePackage(&pkg)
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

		if state.Verbose >= VerboseInfo {
			log.Printf("CreatePackage(%s): origin(%s) destination(%s) method(%s) distance(%f)",
				pkg.PackageID.String()[:8],
				AvroPoint(origin.Position),
				AvroPoint(destination.Position),
				method, distance)
		}

		TriggerArrivalScan(state, &state.Trackers[len(state.Trackers)-1])
	}

}

// UpdatePosition computes a new position for the tracker based on it's current
// position, speed, next location, and the time that has passed.
// Returns: true if we reached the next location, false otherwise
func UpdatePosition(state *State, t *Tracker, delta time.Duration) bool {
	if t.State != enum.InTransit {
		log.Panicf("UpdatePosition can only be called when State == InTransit")
	}

	distanceRemaining := planar.Distance(t.Position, t.NextLocationPosition)

	// calculate the maximum distance this package could have gone in the time
	// that has passed based on this package's current speed
	maxDistance := float64(t.SpeedKMPH*1000) * delta.Hours()

	reachedDestination := false

	if maxDistance >= distanceRemaining {
		// we reached our destination!
		t.Position = t.NextLocationPosition
		reachedDestination = true
	} else {
		percent := maxDistance / distanceRemaining
		t.Position = orb.Point{
			t.Position[0] + percent*(t.NextLocationPosition[0]-t.Position[0]),
			t.Position[1] + percent*(t.NextLocationPosition[1]-t.Position[1]),
		}
	}

	if state.Verbose >= VerboseSilly {
		log.Printf("UpdatePosition(%s): loc(%s) target(%s) travelled(%gkm) remaining(%gkm)",
			t.PackageID.String()[:8],
			AvroPoint(t.Position),
			AvroPoint(t.NextLocationPosition),
			math.Min(maxDistance/1000, distanceRemaining/1000),
			distanceRemaining/1000,
		)
	}

	err := state.Topics.WriteLocation(state.Clock.Now(), t)
	if err != nil {
		log.Panicf("failed to write location to topic: %v", err)
	}

	return reachedDestination
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

	distanceToNext := planar.Distance(currentLocation.Position, nextLocation.Position) / 1000
	speed := state.AvgLandSpeedKMPH
	if distanceToNext > state.MinAirFreightDistanceKM {
		speed = state.AvgAirSpeedKMPH
	}

	// update tracker state fields
	t.State = enum.InTransit
	t.Seq = t.Seq + 1
	t.LastLocationID = currentLocation.LocationID

	// update tracker InTransit fields
	t.SpeedKMPH = int(speed)
	t.Position = currentLocation.Position
	t.NextLocationID = nextLocation.LocationID
	t.NextLocationPosition = nextLocation.Position

	if state.Verbose >= VerboseInfo {
		log.Printf("DepartureScan(%s): loc(%s) speed(%d) target(%s) dist(%f)",
			t.PackageID.String()[:8],
			AvroPoint(currentLocation.Position),
			t.SpeedKMPH,
			AvroPoint(nextLocation.Position),
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

	if state.Verbose >= VerboseInfo {
		currentLocation, err := state.Locations.Lookup(t.LastLocationID)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("ArrivalScan(%s): loc(%s) nextTransition(%s)",
			t.PackageID.String()[:8],
			AvroPoint(currentLocation.Position),
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

	if state.Verbose >= VerboseInfo {
		currentLocation, err := state.Locations.Lookup(t.LastLocationID)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("Delivered(%s): loc(%s)",
			t.PackageID.String()[:8],
			AvroPoint(currentLocation.Position))
	}

	err := state.Topics.WriteTransition(state.Clock.Now(), enum.Delivered, t)
	if err != nil {
		log.Panicf("failed to write transition to topic: %v", err)
	}
}
