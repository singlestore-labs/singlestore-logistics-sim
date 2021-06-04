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

	Verbose int

	MaxPackages                      int
	MaxDelivered                     int
	PackagesPerTick                  *distuv.Normal
	HoursAtRest                      *distuv.Normal
	ProbabilityExpress               float64
	MinShippingDistanceMetres        float64
	MinShippingDistanceMetresSquared float64
	MinAirFreightDistanceMetres      float64
	AvgLandSpeedMetreHours           float64
	AvgAirSpeedMetreHours            float64
}

func NewState(c *Config, locations *LocationIndex, initialTrackers []Tracker) *State {
	return &State{
		Clock:     NewClock(c.StartTime, c.TickDuration),
		Trackers:  initialTrackers,
		Locations: locations,

		Verbose: c.Verbose,

		MaxPackages:                      c.MaxPackages,
		MaxDelivered:                     c.MaxDelivered,
		PackagesPerTick:                  c.PackagesPerTick.ToDist(),
		HoursAtRest:                      c.HoursAtRest.ToDist(),
		ProbabilityExpress:               c.ProbabilityExpress,
		MinShippingDistanceMetres:        c.MinShippingDistanceMetres,
		MinShippingDistanceMetresSquared: c.MinShippingDistanceMetres * c.MinShippingDistanceMetres,
		MinAirFreightDistanceMetres:      c.MinAirFreightDistanceMetres,
		AvgLandSpeedMetreHours:           c.AvgLandSpeedMetreHours,
		AvgAirSpeedMetreHours:            c.AvgAirSpeedMetreHours,
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
			// we only deliver packages which travel farther than MinShippingDistanceMetres
			return planar.DistanceSquared(origin.Position, candidate.Position) > state.MinShippingDistanceMetresSquared
		})
		if err != nil {
			log.Panic(err)
		}

		// extremely crude delivery estimate calculation
		distance := planar.Distance(origin.Position, destination.Position)
		avgSpeed := state.AvgLandSpeedMetreHours
		if method == enum.Express {
			avgSpeed = state.AvgAirSpeedMetreHours
		}
		// include an overhead buffer of 20% due to processing delays per transit point
		hours := (distance / avgSpeed) * 1.2
		deliveryEstimate := now.Add(time.Duration(hours) * time.Hour)

		pkg := Package{
			PackageID:             uuid.NewV4(),
			Received:              now,
			OriginLocationID:      origin.LocationID,
			DestinationLocationID: destination.LocationID,
			DeliveryEstimate:      deliveryEstimate,
			Method:                method,
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
			log.Printf("CreatePackage(%s): origin(%f, %f) destination(%f, %f) method(%s) distance(%f)",
				pkg.PackageID.String()[:8],
				origin.Position.Lon(), origin.Position.Lat(),
				destination.Position.Lon(), destination.Position.Lat(),
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
	maxDistance := float64(t.SpeedKPH*1000) * delta.Hours()

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

	// TODO: submit location to locations topic
	if state.Verbose >= VerboseSilly {
		log.Printf("UpdatePosition(%s): loc(%f, %f) target(%f, %f) travelled(%f)",
			t.PackageID.String()[:8],
			t.Position.Lon(), t.Position.Lat(),
			t.NextLocationPosition.Lon(), t.NextLocationPosition.Lat(),
			math.Min(maxDistance, distanceRemaining))
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

	distanceToNext := planar.Distance(currentLocation.Position, nextLocation.Position)
	speed := state.AvgLandSpeedMetreHours
	if distanceToNext > state.MinAirFreightDistanceMetres {
		speed = state.AvgAirSpeedMetreHours
	}

	// update tracker state fields
	t.State = enum.InTransit
	t.Seq = t.Seq + 1
	t.LastLocationID = currentLocation.LocationID

	// update tracker InTransit fields
	t.SpeedKPH = int(speed / 1000)
	t.Position = currentLocation.Position
	t.NextLocationID = nextLocation.LocationID
	t.NextLocationPosition = nextLocation.Position

	// TODO: submit departure scan to transition topic
	if state.Verbose >= VerboseInfo {
		log.Printf("DepartureScan(%s): loc(%f, %f) speed(%d) target(%f, %f) dist(%f)",
			t.PackageID.String()[:8],
			currentLocation.Position.Lon(), currentLocation.Position.Lat(),
			t.SpeedKPH,
			nextLocation.Position.Lon(), nextLocation.Position.Lat(),
			distanceToNext)
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

	// TODO: submit arrival scan to transition topic
	if state.Verbose >= VerboseInfo {
		currentLocation, err := state.Locations.Lookup(t.LastLocationID)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("ArrivalScan(%s): loc(%f, %f) nextTransition(%s)",
			t.PackageID.String()[:8],
			currentLocation.Position.Lon(), currentLocation.Position.Lat(),
			t.NextTransitionTime.Sub(now))
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

	// TODO: submit delivered to transition topic
	if state.Verbose >= VerboseInfo {
		currentLocation, err := state.Locations.Lookup(t.LastLocationID)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("Delivered(%s): loc(%f, %f)",
			t.PackageID.String()[:8],
			currentLocation.Position.Lon(), currentLocation.Position.Lat())
	}
}
