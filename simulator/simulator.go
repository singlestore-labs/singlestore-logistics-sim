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

type State struct {
	Clock     *Clock
	Trackers  []Tracker
	Locations *LocationGraph

	PackagesPerTick    distuv.Normal
	HoursAtRest        distuv.Normal
	ProbabilityExpress float64
}

func NewState(c *Config, locations *LocationGraph, initialTrackers []Tracker) *State {
	return &State{
		Clock:    NewClock(c.StartTime, c.TickDuration),
		Trackers: initialTrackers,
		PackagesPerTick: distuv.Normal{
			Mu:    c.AvgPackagesPerTick,
			Sigma: c.StddevPackagesPerTick,
		},
		HoursAtRest: distuv.Normal{
			Mu:    c.AvgHoursAtRest,
			Sigma: c.StddevHoursAtRest,
		},
		ProbabilityExpress: c.ProbabilityExpress,
	}
}

func Simulate(state *State) {
	for {
		delta := state.Clock.Tick()
		now := state.Clock.Now()

		// approx how many new packages should been created
		numNewPackages := int(math.Round(state.PackagesPerTick.Rand()))

		// create new packages
		for i := 0; i < numNewPackages; i++ {
			method := enum.Standard
			if rand.Float64() > state.ProbabilityExpress {
				method = enum.Express
			}

			pkg := Package{
				PackageID:             uuid.NewV4(),
				Received:              now,
				OriginLocationID:      0,   // TODO: select random location
				DestinationLocationID: 0,   // TODO: select random location
				DeliveryEstimate:      now, // TODO: compute delivery estimate
				Method:                method,
			}

			nextTransitionTime := now.Add(time.Hour * time.Duration(state.HoursAtRest.Rand()))

			state.Trackers = append(state.Trackers, Tracker{
				PackageID:             pkg.PackageID,
				Method:                pkg.Method,
				DestinationLocationID: pkg.DestinationLocationID,

				// packages always start at rest
				State:          enum.AtRest,
				Seq:            0,
				LastLocationID: pkg.OriginLocationID,

				NextTransitionTime: nextTransitionTime,
			})
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
				if now.After(tracker.NextTransitionTime) {
					TriggerDepartureScan(tracker)
				}

			case enum.InTransit:
				if UpdatePosition(tracker, delta) {
					// the package has reached it's current destination
					if tracker.DestinationLocationID == tracker.NextLocationID {
						// the package has reached it's final destination
						TriggerDelivered(tracker)
						numDelivered++
					} else {
						// the package has reached a interim destination
						TriggerArrivalScan(tracker)
					}
				} else {
					// the package is still in transit
					TrackLocation(tracker)
				}

			default:
				log.Panicf("unknown state %+v for package %s", tracker.State, tracker.PackageID)
			}
		}

	}
}

// UpdatePosition computes a new position for the tracker based on it's current
// position, speed, next location, and the time that has passed.
// Returns: true if we reached the next location, false otherwise
func UpdatePosition(t *Tracker, delta time.Duration) bool {
	if t.State != enum.InTransit {
		log.Panicf("UpdatePosition can only be called when State == InTransit")
	}

	distanceRemaining := geo.Distance(t.Position, t.NextLocationPosition)

	// calculate the maximum distance this package could have gone in the time
	// that has passed based on this package's current speed
	maxDistance := float64(t.SpeedKPH*1000) * delta.Hours()

	if maxDistance >= distanceRemaining {
		// we reached our destination!
		t.Position = t.NextLocationPosition
		return true
	}

	percent := (distanceRemaining - maxDistance) / distanceRemaining
	t.Position = orb.Point{
		t.Position[0] + percent*(t.NextLocationPosition[0]*t.Position[0]),
		t.Position[1] + percent*(t.NextLocationPosition[1]*t.Position[1]),
	}

	return false
}

func TriggerDepartureScan(t *Tracker) {
	if t.State != enum.AtRest {
		log.Panicf("TriggerDepartureScan can only be called when State == AtRest")
	}

	/*
		need to determine the next location
			express -> pick the next closest hub (or final destination if closer)
			point -> pick the next closest location

		compute speed -> distance > X km then AIR else LAND

		submit departure scan transition
		update tracker
	*/
}

func TrackLocation(t *Tracker) {
	if t.State != enum.InTransit {
		log.Panicf("TriggerDepartureScan can only be called when State == InTransit")
	}

	/*
		submit location to tracking topic
		update tracker
	*/
}

func TriggerArrivalScan(t *Tracker) {
	if t.State != enum.InTransit {
		log.Panicf("TriggerArrivalScan can only be called when State == InTransit")
	}

	/*
		submit arrival scan transition
		update tracker
	*/
}

func TriggerDelivered(t *Tracker) {
	if t.State != enum.InTransit {
		log.Panicf("TriggerDelivered can only be called when State == InTransit")
	}

	/*
		submit delivered transition
		update tracker
	*/
}
