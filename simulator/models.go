package simulator

import (
	"time"

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
