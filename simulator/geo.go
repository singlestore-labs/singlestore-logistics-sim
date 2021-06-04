package simulator

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/project"
)

var (
	minGeoPoint = project.WGS84.ToMercator(orb.Point{-180, -90})
	maxGeoPoint = project.WGS84.ToMercator(orb.Point{180, 90})
)

func randFloatInRange(min float64, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func randGeoPoint() orb.Point {
	return orb.Point{
		randFloatInRange(minGeoPoint[0], maxGeoPoint[0]),
		randFloatInRange(minGeoPoint[1], maxGeoPoint[1]),
	}
}

func NewPointFromWGS84(lon, lat float64) orb.Point {
	return project.WGS84.ToMercator(orb.Point{lon, lat})
}

// AvroPoint is defined for cases where we need to serialize a Point to Avro
type AvroPoint orb.Point

func (p AvroPoint) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p AvroPoint) String() string {
	out := project.Mercator.ToWGS84(orb.Point(p))
	// would prefer to use %g but singlestore doesn't support scientific precision in numbers
	// using %f results in fixed width floats
	return fmt.Sprintf("POINT(%s %s)", strconv.FormatFloat(out[0], 'f', -1, 64), strconv.FormatFloat(out[1], 'f', -1, 64))
}
