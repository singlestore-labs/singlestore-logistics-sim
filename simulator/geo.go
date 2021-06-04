package simulator

import (
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
