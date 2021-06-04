package enum

type DeliveryMethod string

const (
	Standard DeliveryMethod = "standard"
	Express  DeliveryMethod = "express"
)

type TransitionKind string

const (
	ArrivalScan   TransitionKind = "arrival_scan"
	DepartureScan TransitionKind = "departure_scan"
	Delivered     TransitionKind = "delivered"
)

type PackageState string

const (
	AtRest    PackageState = "at_rest"
	InTransit PackageState = "in_transit"
)

type LocationKind string

const (
	Hub   LocationKind = "hub"
	Point LocationKind = "point"
)
