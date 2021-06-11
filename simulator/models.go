package simulator

import (
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	uuid "github.com/satori/go.uuid"

	"simulator/enum"
)

type Package struct {
	PackageID             uuid.UUID
	SimulatorID           string
	Received              time.Time
	OriginLocationID      int64
	DestinationLocationID int64
	DeliveryEstimate      time.Time
	Method                enum.DeliveryMethod
}

type Transition struct {
	PackageID      uuid.UUID
	Seq            int
	LocationID     int64
	NextLocationID int64
	Recorded       time.Time
	Kind           enum.TransitionKind
}

// LocationRecord is only used when writing to the locations topic
type LocationRecord struct {
	PackageID uuid.UUID
	Recorded  time.Time
	Position  AvroPoint
}

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

	// The following fields are updated when a package is at rest
	NextTransitionTime time.Time

	// The following fields are updated when a package is in transit

	// SpeedKMPH tracks the package's current speed (kilometres/h) while in transit
	SpeedKMPH int
	// Position tracks the package's current location while in transit
	Position orb.Point
	// NextLocationID tracks the package's next location
	NextLocationID int64
	// NextLocationPosition tracks the package's next location's position
	NextLocationPosition orb.Point
}

func NewTrackersFromActivePackages(c *Config, l *LocationIndex, packages []DBActivePackage) ([]Tracker, error) {
	out := make([]Tracker, 0, len(packages))
	for _, pkg := range packages {
		pkgState := enum.AtRest
		if pkg.TransitionKind == enum.DepartureScan {
			pkgState = enum.InTransit
		}

		currentPosition := NewPointFromWGS84(pkg.Longitude, pkg.Latitude)

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

		t := Tracker{
			PackageID:             pkg.PackageID,
			Method:                pkg.Method,
			DestinationLocationID: pkg.DestinationLocationID,

			Delivered:      false,
			State:          pkgState,
			Seq:            pkg.TransitionSeq,
			LastLocationID: pkg.TransitionLocationID,

			// we don't set nextTransitionTime which will cause the package to
			// immediately transition if it is currently AtRest

			SpeedKMPH:            int(speed),
			Position:             currentPosition,
			NextLocationID:       pkg.TransitionNextLocationID,
			NextLocationPosition: segmentEnd.Position,
		}
		out = append(out, t)
	}

	return out, nil
}
