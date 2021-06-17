package simulator

import (
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

func NewTrackersFromActivePackages(c *Config, l *LocationIndex, packages []DBActivePackage) ([]Tracker, error) {
	out := make([]Tracker, 0, len(packages))
	for _, pkg := range packages {
		pkgState := enum.AtRest
		if pkg.TransitionKind == enum.DepartureScan {
			pkgState = enum.InTransit
		}

		var nextTransitionTime time.Time

		if pkgState == enum.InTransit {
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

		t := Tracker{
			PackageID:             pkg.PackageID,
			Method:                pkg.Method,
			DestinationLocationID: pkg.DestinationLocationID,

			Delivered:      false,
			State:          pkgState,
			Seq:            pkg.TransitionSeq,
			LastLocationID: pkg.TransitionLocationID,

			NextTransitionTime: nextTransitionTime,
			NextLocationID:     pkg.TransitionNextLocationID,
		}
		out = append(out, t)
	}

	return out, nil
}
