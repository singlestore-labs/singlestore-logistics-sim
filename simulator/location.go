package simulator

import (
	"container/list"
	"simulator/enum"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/quadtree"
	"github.com/pkg/errors"
)

var (
	nearestCount = 3
	empty        = struct{}{}
)

type Location struct {
	LocationID int64
	Kind       enum.LocationKind
	Position   orb.Point
	Nearest    []*Location
}

// satisfy the geo.Pointer interface
func (l *Location) Point() orb.Point {
	return l.Position
}

func NewLocationFromDB(dbloc DBLocation) *Location {
	return &Location{
		LocationID: dbloc.LocationID,
		Kind:       dbloc.Kind,
		Position:   NewPointFromWGS84(dbloc.Longitude, dbloc.Latitude),
	}
}

type LocationQueue struct {
	list.List
}

func NewLocationQueue() *LocationQueue {
	q := LocationQueue{}
	q.Init()
	return &q
}

func (q *LocationQueue) Pop() *Location {
	out := q.List.Front()
	q.List.Remove(out)
	return out.Value.(*Location)
}

type LocationIndex struct {
	qt *quadtree.Quadtree
	ht map[int64]*Location
}

func NewLocationIndexFromDB(dblocs []DBLocation) (*LocationIndex, error) {
	idx := &LocationIndex{
		qt: quadtree.New(orb.Bound{Min: minGeoPoint, Max: maxGeoPoint}),
		ht: make(map[int64]*Location),
	}
	for _, dbloc := range dblocs {
		loc := NewLocationFromDB(dbloc)
		err := idx.qt.Add(loc)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		idx.ht[loc.LocationID] = loc
	}

	// precompute and store knearest for each location
	buf := make([]orb.Pointer, 0, nearestCount)
	for _, loc := range idx.ht {
		nearest := idx.qt.KNearestMatching(buf, loc.Position, nearestCount, func(p orb.Pointer) bool {
			return p != loc
		})

		loc.Nearest = make([]*Location, nearestCount)
		for i, n := range nearest {
			loc.Nearest[i] = n.(*Location)
		}
	}

	return idx, nil
}

func (i *LocationIndex) NextLocation(current *Location, destination *Location, method enum.DeliveryMethod) *Location {
	// our current squared distance to the destination
	currentToDestination := planar.DistanceSquared(current.Position, destination.Position)

	// 200km squared
	minDistanceSquared := float64(200 * 1000 * 200 * 1000)

	seen := make(map[int64]struct{})
	seen[current.LocationID] = empty

	q := NewLocationQueue()
	q.PushBack(current)

	for q.Len() > 0 {
		candidate := q.Pop()

		// queue unseen neighbors
		for _, neighbor := range candidate.Nearest {
			if _, ok := seen[neighbor.LocationID]; !ok {
				seen[neighbor.LocationID] = empty
				q.PushBack(neighbor)
			}
		}

		if candidate == current {
			continue
		}

		// if one of the nearest locations is our destination, select it
		if candidate.LocationID == destination.LocationID {
			return candidate
		}

		// if we are express shipping then only consider hubs
		if method == enum.Express && candidate.Kind != enum.Hub {
			continue
		}

		// get the distance from our current location to the candidate
		currentToCandidate := planar.DistanceSquared(current.Position, candidate.Position)

		// only select destinations at least 200km away
		if currentToCandidate < minDistanceSquared {
			continue
		}

		// get the distance from the candidate location to our destination
		candidateToDestination := planar.DistanceSquared(candidate.Position, destination.Position)

		// make sure we always move towards our destination
		if candidateToDestination < currentToDestination {
			return candidate
		}
	}

	// if we fail to find the next nearest location - just send the package directly to the destination
	return destination
}

func (i *LocationIndex) Lookup(locationID int64) (*Location, error) {
	if l, ok := i.ht[locationID]; ok {
		return l, nil
	}
	return nil, errors.Errorf("location %d not found", locationID)
}

// Rand returns a random location in the index for which the filter returns true
// filter can be nil which implies no filter
func (i *LocationIndex) Rand(filter quadtree.FilterFunc) (*Location, error) {
	ptr := i.qt.Matching(randGeoPoint(), filter)
	if ptr == nil {
		return nil, errors.New("calling Rand() on an empty LocationIndex or filter is too restrictive")
	}
	return ptr.(*Location), nil
}
