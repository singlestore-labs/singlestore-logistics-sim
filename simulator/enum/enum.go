package enum

type DeliveryMethod string

const (
	Standard DeliveryMethod = "standard"
	Express  DeliveryMethod = "express"
)

type TransitionKind string

const (
	ArrivalScan   TransitionKind = "arrival scan"
	DepartureScan TransitionKind = "departure scan"
	Delivered     TransitionKind = "delivered"
)

type PackageState string

const (
	AtRest    PackageState = "at rest"
	InTransit PackageState = "in transit"
)

type LocationKind string

const (
	Hub   LocationKind = "hub"
	Point LocationKind = "point"
)
