package simulator

import (
	"simulator/enum"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/orb/quadtree"
	"github.com/pkg/errors"
)

type Location struct {
	LocationID int64
	Kind       enum.LocationKind
	Position   orb.Point
}

// satisfy the geo.Pointer interface
func (l *Location) Point() orb.Point {
	return l.Position
}

func NewLocationFromDB(dbloc DBLocation) *Location {
	return &Location{
		LocationID: dbloc.LocationID,
		Kind:       dbloc.Kind,
		Position:   orb.Point{dbloc.Longitude, dbloc.Latitude},
	}
}

type LocationIndex struct {
	qt *quadtree.Quadtree
}

func (i *LocationIndex) NextLocation(current *Location, destination *Location, method enum.DeliveryMethod) (*Location, error) {

	// THE FOLLOWING CODE IS FAR FROM RIGHT
	/*
		we need to ensure that we are generally moving *towards* the destination
		i.e. this code can pick a location in the wrong direction

		we also need to ensure that we can't pick a cycle of points - this might be easiest if we are always moving towards the destination

		IDEA!
		we can restrict ourself to locations which are between us an the destination via the match function below
	*/

	var nearest []orb.Pointer
	if method == enum.Express {
		// pick the next closest hub (or final destination if closer)
		nearest = i.qt.KNearestMatching(nil, current.Position, 1, func(p orb.Pointer) bool {
			loc := p.(*Location)
			return loc.Kind == enum.Hub || loc.LocationID == destination.LocationID
		})
	} else {
		// pick the next closest location
		nearest = i.qt.KNearest(nil, current.Position, 1)
	}
	if len(nearest) == 0 {
		return nil, errors.Errorf("failed to find next location; current=%s, destination=%s, method=%s", current.LocationID, destination.LocationID, method)
	}
	return nearest[0].(*Location), nil
}

// TODO:
/*
	function to determine the next location
		express -> pick the next closest hub (or final destination if closer)
		point -> pick the next closest location

	function to lookup a location's position

	function to get a random location
*/

// Distance returns the distance in metres between two locations
func Distance(a, b *Location) float64 {
	return geo.Distance(a.Position, b.Position)
}
