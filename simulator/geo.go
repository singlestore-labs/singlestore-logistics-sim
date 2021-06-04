package simulator

import (
	"fmt"
	"math/rand"

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
	return fmt.Sprintf("POINT(%g %g)", out[0], out[1])
}
