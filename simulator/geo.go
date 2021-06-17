package simulator

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/project"
)

var (
	// We don't consider the last 6 deg at the poles since Mercator projects them to infinity
	minGeoPoint = project.WGS84.ToMercator(orb.Point{-180, -84})
	maxGeoPoint = project.WGS84.ToMercator(orb.Point{180, 84})

	mercatorWidth  = maxGeoPoint[0] * 2
	mercatorHeight = maxGeoPoint[1] * 2

	// hand drawn, far from accurate but good enough
	majorPopulationCenters = mustParsePopulationCenters("MULTIPOLYGON(((-125.15625000000001 25.48295117535531,-79.1015625 25.48295117535531,-79.1015625 49.15296965617042,-125.15625000000001 49.15296965617042,-125.15625000000001 25.48295117535531)),((-133.59375 49.38237278700955,-54.66796875 49.38237278700955,-54.66796875 56.65622649350222,-133.59375 56.65622649350222,-133.59375 49.38237278700955)),((-79.1015625 44.96479793033101,-52.55859375 44.96479793033101,-52.55859375 49.26780455063753,-79.1015625 49.26780455063753,-79.1015625 44.96479793033101)),((-78.92578124999999 34.88593094075317,-70.6640625 34.88593094075317,-70.6640625 44.715513732021336,-78.92578124999999 44.715513732021336,-78.92578124999999 34.88593094075317)),((-108.45703125 10.487811882056695,-74.53125 10.487811882056695,-74.53125 25.005972656239187,-108.45703125 25.005972656239187,-108.45703125 10.487811882056695)),((-85.25390625 -13.068776734357694,-35.859375 -13.068776734357694,-35.859375 9.795677582829743,-85.25390625 9.795677582829743,-85.25390625 -13.068776734357694)),((-75.41015624999999 -23.563987128451217,-38.67187499999999 -23.563987128451217,-38.67187499999999 -13.581920900545844,-75.41015624999999 -13.581920900545844,-75.41015624999999 -23.563987128451217)),((-72.59765625 -37.16031654673676,-52.3828125 -37.16031654673676,-52.3828125 -24.046463999666567,-72.59765625 -24.046463999666567,-72.59765625 -37.16031654673676)),((-164.35546875 59.355596110016315,-140.2734375 59.355596110016315,-140.2734375 63.39152174400882,-164.35546875 63.39152174400882,-164.35546875 59.355596110016315)),((-159.9609375 17.97873309555617,-153.6328125 17.97873309555617,-153.6328125 22.59372606392931,-159.9609375 22.59372606392931,-159.9609375 17.97873309555617)),((-194.23828125 -47.5172006978394,-180.52734375 -47.5172006978394,-180.52734375 -35.7465122599185,-194.23828125 -35.7465122599185,-194.23828125 -47.5172006978394)),((-217.6171875 -43.96119063892024,-207.0703125 -43.96119063892024,-207.0703125 -10.487811882056683,-217.6171875 -10.487811882056683,-217.6171875 -43.96119063892024)),((-249.2578125 -36.738884124394296,-241.69921874999997 -36.738884124394296,-241.69921874999997 -17.811456088564473,-249.2578125 -17.811456088564473,-249.2578125 -36.738884124394296)),((-266.30859375 -8.754794702435618,-223.06640625 -8.754794702435618,-223.06640625 22.917922936146045,-266.30859375 22.917922936146045,-266.30859375 -8.754794702435618)),((-377.2265625 21.289374355860424,-214.98046875 21.289374355860424,-214.98046875 49.95121990866204,-377.2265625 49.95121990866204,-377.2265625 21.289374355860424)),((-378.28125 4.390228926463396,-301.2890625 4.390228926463396,-301.2890625 20.632784250388028,-378.28125 20.632784250388028,-378.28125 4.390228926463396)),((-351.03515625 -35.31736632923786,-319.921875 -35.31736632923786,-319.921875 3.8642546157214084,-351.03515625 3.8642546157214084,-351.03515625 -35.31736632923786)),((-317.28515625 -25.324166525738384,-310.078125 -25.324166525738384,-310.078125 -10.660607953624762,-317.28515625 -10.660607953624762,-317.28515625 -25.324166525738384)),((-290.21484375 5.965753671065536,-272.4609375 5.965753671065536,-272.4609375 21.12549763660628,-290.21484375 21.12549763660628,-290.21484375 5.965753671065536)),((-371.07421875 49.724479188712984,-357.5390625 49.724479188712984,-357.5390625 59.44507509904714,-371.07421875 59.44507509904714,-371.07421875 49.724479188712984)),((-357.5390625 49.724479188712984,-324.31640625 49.724479188712984,-324.31640625 69.77895177646761,-357.5390625 69.77895177646761,-357.5390625 49.724479188712984)),((-323.96484375 49.95121990866204,-272.109375 49.95121990866204,-272.109375 65.2198939361321,-323.96484375 65.2198939361321,-323.96484375 49.95121990866204)))")
)

type PopulationCenter struct {
	bound orb.Bound
	area  float64
}

func mustParsePopulationCenters(s string) []PopulationCenter {
	// parse wkt
	p, err := wkt.UnmarshalMultiPolygon(s)
	if err != nil {
		panic(err)
	}

	// some of the points are wrapped horizontally, lets fix them
	p = project.MultiPolygon(p, func(i orb.Point) orb.Point {
		if i[0] < -180 {
			i[0] += 360
		}
		return i
	})

	// project to Mercator
	p = project.MultiPolygon(p, project.WGS84.ToMercator)

	// convert to bounds
	out := make([]PopulationCenter, 0, len(p))
	for _, poly := range p {
		out = append(out, PopulationCenter{
			bound: poly.Bound(),
			area:  planar.Area(poly),
		})
	}

	// sort the population centers by area
	sort.Slice(out, func(i, j int) bool {
		return out[i].area < out[j].area
	})

	return out
}

// normalizePoint converts a mercator point to a point in the bounds
// (0, 0) -> (1, 1)
func normalizePoint(p orb.Point) orb.Point {
	return orb.Point{
		(p[0] + (mercatorWidth / 2)) / mercatorWidth,
		(p[1] + (mercatorHeight / 2)) / mercatorHeight,
	}
}

func randFloatInRange(min float64, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// randGeoPoint picks a random point on the globe usually(*) within a major population center
// (*) for performance we generate points within the bounds of each polygon
func randGeoPoint() orb.Point {
	numPopulationCenters := len(majorPopulationCenters)

	// pick a random area
	randArea := randFloatInRange(
		majorPopulationCenters[0].area,
		majorPopulationCenters[numPopulationCenters-1].area,
	)

	// pick bounds from population centers weighted by area
	idx := sort.Search(numPopulationCenters, func(i int) bool {
		return majorPopulationCenters[i].area >= randArea
	})

	bounds := majorPopulationCenters[idx].bound

	// generate a point within bounds
	return orb.Point{
		randFloatInRange(bounds.Min[0], bounds.Max[0]),
		randFloatInRange(bounds.Min[1], bounds.Max[1]),
	}
}

func NewPointFromWGS84(lon, lat float64) orb.Point {
	return project.WGS84.ToMercator(orb.Point{lon, lat})
}

// AvroPoint is defined for cases where we need to serialize a Point to Avro
type AvroPoint orb.Point

func (p AvroPoint) MarshalText() ([]byte, error) {
	out := project.Mercator.ToWGS84(orb.Point(p))
	// would prefer to use %g but singlestore doesn't support scientific precision in numbers
	// using %f results in fixed width floats
	return []byte(fmt.Sprintf("POINT(%s %s)", strconv.FormatFloat(out[0], 'f', 10, 64), strconv.FormatFloat(out[1], 'f', 10, 64))), nil
}

func (p AvroPoint) String() string {
	out := project.Mercator.ToWGS84(orb.Point(p))
	return fmt.Sprintf("(%f %f)", out[0], out[1])
}
