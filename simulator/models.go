package simulator

import (
	"time"

	"github.com/paulmach/orb"
	uuid "github.com/satori/go.uuid"

	"simulator/enum"
)

type Package struct {
	PackageID             uuid.UUID
	Received              time.Time
	OriginLocationID      int64
	DestinationLocationID int64
	DeliveryEstimate      time.Time
	Method                enum.DeliveryMethod
}

type Tracker struct {
	PackageID             uuid.UUID
	Method                enum.DeliveryMethod
	DestinationLocationID int64
	Delivered             bool

	State          enum.PackageState
	Seq            int
	LastLocationID int64

	// The following fields are only used for packages at rest (State == AtRest)
	NextTransitionTime time.Time

	// The following fields are only used for packages in transit (State == InTransit)

	// SpeedKPH tracks the package's current speed (kilometres per hour) while in transit
	SpeedKPH int
	// Position tracks the package's current location while in transit
	Position orb.Point
	// NextLocationID tracks the package's next location
	NextLocationID int64
	// NextLocationPosition tracks the package's next location's position
	NextLocationPosition orb.Point
}
