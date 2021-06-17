package simulator

import (
	"fmt"

	"github.com/paulmach/orb"
)

var (
	minGeoPoint = orb.Point{-180, -90}
	maxGeoPoint = orb.Point{180, 90}
)

func PointString(p orb.Point) string {
	return fmt.Sprintf("(%f %f)", p[0], p[1])
}
