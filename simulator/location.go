package simulator

import (
	"container/heap"
	"simulator/enum"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/quadtree"
	"github.com/pkg/errors"
)

var (
	nearestCount = 5
	empty        = struct{}{}

	// 200km squared
	minDistanceSquared = float64(200 * 1000 * 200 * 1000)
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

type LocationQueueItem struct {
	loc                   *Location
	distanceToDestination float64
}

type LocationQueue []*LocationQueueItem

var _ heap.Interface = &LocationQueue{}

func NewLocationQueue() LocationQueue {
	return LocationQueue(make(LocationQueue, 0, nearestCount*4))
}

func (l LocationQueue) Len() int { return len(l) }
func (l LocationQueue) Less(i, j int) bool {
	return l[i].distanceToDestination > l[j].distanceToDestination
}
func (l LocationQueue) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

// add x as element Len()
func (l *LocationQueue) Push(x interface{}) {
	*l = append(*l, x.(*LocationQueueItem))
}

// remove and return element Len() - 1
func (l *LocationQueue) Pop() interface{} {
	old := *l
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*l = old[0 : n-1]
	return item
}

func (l *LocationQueue) PushLocation(loc *Location, distanceToDestination float64) {
	heap.Push(l, &LocationQueueItem{
		loc:                   loc,
		distanceToDestination: distanceToDestination,
	})
}

func (l *LocationQueue) PopLocation() (*Location, float64) {
	item := heap.Pop(l).(*LocationQueueItem)
	return item.loc, item.distanceToDestination
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

	seen := make(map[int64]struct{})
	seen[current.LocationID] = empty

	q := NewLocationQueue()
	q.PushLocation(current, 0)

	for q.Len() > 0 {
		candidate, distanceToDestination := q.PopLocation()

		// prepend unseen neighbors based on distance to destination
		for _, neighbor := range candidate.Nearest {
			if _, ok := seen[neighbor.LocationID]; !ok {
				seen[neighbor.LocationID] = empty

				// if one of the neighbors is our destination, select it
				if candidate.LocationID == destination.LocationID {
					return candidate
				}

				// get the distance from the neighbor location to our destination
				neighborToDestination := planar.DistanceSquared(neighbor.Position, destination.Position)
				q.PushLocation(current, neighborToDestination)

			}
		}

		if candidate == current {
			continue
		}

		// make sure we only consider candidates who are closer to the destination than we are
		if distanceToDestination >= currentToDestination {
			continue
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

		// we found our match
		return candidate
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
