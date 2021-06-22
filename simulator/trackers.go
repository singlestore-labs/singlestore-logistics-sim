package simulator

import (
	"container/heap"
	"simulator/enum"
	"time"

	"github.com/paulmach/orb/planar"
	uuid "github.com/satori/go.uuid"
)

type Tracker struct {
	// The following fields should never change
	PackageID             uuid.UUID
	Method                enum.DeliveryMethod
	DestinationLocationID int64

	// The following fields may be updated on each transition
	Delivered      bool
	State          enum.PackageState
	Seq            int
	LastLocationID int64

	NextTransitionTime time.Time
	NextLocationID     int64
}

type Trackers []*Tracker

var _ heap.Interface = &Trackers{}

func NewTrackersFromActivePackages(c *Config, l *LocationIndex, packages []DBActivePackage) (Trackers, error) {
	out := make(Trackers, 0, len(packages))

	for _, pkg := range packages {
		var nextTransitionTime time.Time

		if pkg.StateKind == enum.InTransit {
			segmentStart, err := l.Lookup(pkg.TransitionLocationID)
			if err != nil {
				return nil, err
			}
			segmentEnd, err := l.Lookup(pkg.TransitionNextLocationID)
			if err != nil {
				return nil, err
			}

			segmentDistance := planar.Distance(segmentStart.Position, segmentEnd.Position) / 1000
			speed := c.AvgLandSpeedKMPH
			if segmentDistance > c.MinAirFreightDistanceKM {
				speed = c.AvgAirSpeedKMPH
			}

			duration := time.Hour * time.Duration(segmentDistance/speed)
			nextTransitionTime = pkg.TransitionRecorded.Add(duration)
		}

		out = append(out, &Tracker{
			PackageID:             pkg.PackageID,
			Method:                pkg.Method,
			DestinationLocationID: pkg.DestinationLocationID,

			Delivered:      false,
			State:          pkg.StateKind,
			Seq:            pkg.TransitionSeq,
			LastLocationID: pkg.TransitionLocationID,

			NextTransitionTime: nextTransitionTime,
			NextLocationID:     pkg.TransitionNextLocationID,
		})
	}

	heap.Init(&out)

	return out, nil
}

func (t Trackers) Len() int { return len(t) }
func (t Trackers) Less(i, j int) bool {
	return t[i].NextTransitionTime.Before(t[j].NextTransitionTime)
}
func (t Trackers) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// add x as element Len()
func (t *Trackers) Push(x interface{}) {
	*t = append(*t, x.(*Tracker))
}

// remove and return element Len() - 1
func (t *Trackers) Pop() interface{} {
	old := *t
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*t = old[0 : n-1]
	return item
}

func (t *Trackers) PushTracker(tracker *Tracker) {
	heap.Push(t, tracker)
}

func (t *Trackers) PopTracker() *Tracker {
	return heap.Pop(t).(*Tracker)
}

func (t Trackers) EarliestTransitionTime() time.Time {
	return t[0].NextTransitionTime
}
