package simulator

import (
	"container/heap"
	"fmt"
	"log"
	"math"
	"simulator/enum"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/quadtree"
	"github.com/pkg/errors"
)

var (
	empty = struct{}{}
)

type Location struct {
	LocationID  int64
	Kind        enum.LocationKind
	Position    orb.Point
	Nearest     []*Location
	NearestHubs []*Location
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
	score                 float64
	distanceToDestination float64
}

type LocationQueue []*LocationQueueItem

var _ heap.Interface = &LocationQueue{}

func NewLocationQueue() LocationQueue {
	return LocationQueue(make(LocationQueue, 0))
}

func (l LocationQueue) Len() int { return len(l) }
func (l LocationQueue) Less(i, j int) bool {
	return l[i].score > l[j].score
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

func (l *LocationQueue) PushLocation(loc *Location, score float64, distanceToDestination float64) {
	heap.Push(l, &LocationQueueItem{
		loc:                   loc,
		score:                 score,
		distanceToDestination: distanceToDestination,
	})
}

func (l *LocationQueue) PopLocation() (*Location, float64) {
	item := heap.Pop(l).(*LocationQueueItem)
	return item.loc, item.distanceToDestination
}

type LocationIndex struct {
	qt           *quadtree.Quadtree
	ht           map[int64]*Location
	debugLogging bool
}

func NewLocationIndexFromDB(dblocs []DBLocation, debugLogging bool) (*LocationIndex, error) {
	idx := &LocationIndex{
		qt:           quadtree.New(orb.Bound{Min: minGeoPoint, Max: maxGeoPoint}),
		ht:           make(map[int64]*Location),
		debugLogging: debugLogging,
	}
	for _, dbloc := range dblocs {
		loc := NewLocationFromDB(dbloc)
		err := idx.qt.Add(loc)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		idx.ht[loc.LocationID] = loc
	}

	const (
		Pi      = math.Pi
		PiOver2 = Pi / 2
	)

	locationFilter := func(origin *Location, direction string, kind enum.LocationKind) func(p orb.Pointer) bool {
		normOrigin := normalizePoint(origin.Position)

		return func(p orb.Pointer) bool {
			loc := p.(*Location)
			if loc == origin {
				return false
			}
			if kind != enum.Any && kind != loc.Kind {
				return false
			}

			normLoc := normalizePoint(loc.Position)

			dx := normLoc[0] - normOrigin[0]
			dy := normLoc[1] - normOrigin[1]

			if dx < -0.5 {
				dx += 1
			}
			if dx > 0.5 {
				dx -= 1
			}
			if dy < -0.5 {
				dy += 1
			}
			if dy > 0.5 {
				dy -= 1
			}

			angle := math.Atan2(dy, dx)

			switch direction {
			case "ne":
				return angle > 0 && angle <= PiOver2
			case "nw":
				return angle > PiOver2 && angle <= Pi
			case "sw":
				return angle > -Pi && angle <= -PiOver2
			case "se":
				return angle > -PiOver2 && angle <= 0
			}

			panic(fmt.Sprintf("unknown direction: '%s'", direction))
		}
	}

	// precompute and store knearest for each location per direction (nw, ne, sw, se)
	nearestPerDirection := 2
	directions := []string{"ne", "nw", "sw", "se"}
	buf := make([]orb.Pointer, 0, nearestPerDirection)

	for _, loc := range idx.ht {
		loc.Nearest = make([]*Location, 0, nearestPerDirection*4)
		loc.NearestHubs = make([]*Location, 0, nearestPerDirection*4)

		for _, direction := range directions {
			nearest := idx.qt.KNearestMatching(buf, loc.Position, nearestPerDirection, locationFilter(loc, direction, enum.Any))
			for _, n := range nearest {
				loc.Nearest = append(loc.Nearest, n.(*Location))
			}
			nearestHubs := idx.qt.KNearestMatching(buf, loc.Position, nearestPerDirection, locationFilter(loc, direction, enum.Hub))
			for _, n := range nearestHubs {
				loc.NearestHubs = append(loc.NearestHubs, n.(*Location))
			}
		}
	}

	return idx, nil
}

func (i *LocationIndex) NextLocation(current *Location, destination *Location, method enum.DeliveryMethod) *Location {
	// our current squared distance to the destination
	currentToDestination := planar.DistanceSquared(current.Position, destination.Position)

	// min distance should be at least 1/2 of the remaining distance to the destination
	minDistanceSquared := math.Pow(math.Sqrt(currentToDestination)/2, 2)

	seen := make(map[*Location]struct{})
	seen[current] = empty

	q := NewLocationQueue()
	q.PushLocation(current, 0, currentToDestination)

	if i.debugLogging {
		log.Printf("NextLocation: remaining distance = %0.2fkm", math.Sqrt(currentToDestination)/1000)
	}

	considered := 0
	if i.debugLogging {
		defer func() {
			log.Printf("NextLocation considered %d candidates, remaining queue size: %d", considered, q.Len())
		}()
	}

	for q.Len() > 0 {
		candidate, distanceToDestination := q.PopLocation()
		considered++

		if i.debugLogging {
			log.Printf("NextLocation: candidate distance = %0.2fkm", math.Sqrt(distanceToDestination)/1000)
		}

		nearest := candidate.Nearest
		if method == enum.Express {
			nearest = candidate.NearestHubs
		}

		// prepend unseen neighbors based on distance to destination
		for _, neighbor := range nearest {
			if _, ok := seen[neighbor]; !ok {
				seen[neighbor] = empty

				// get the distance from the neighbor location to our destination
				neighborToDestination := planar.DistanceSquared(neighbor.Position, destination.Position)

				// calculate the diff
				// if diff is negative then we are moving the wrong direction
				// if diff is positive then we are moving the right direction
				diff := currentToDestination - neighborToDestination

				// calculate the location score
				// we prefer neighbors who are towards the destination, but
				// closer to us than they are close to the destination
				score := currentToDestination + diff

				q.PushLocation(neighbor, score, neighborToDestination)
			}
		}

		if candidate == current {
			continue
		}

		// if one of the candidates is our destination, select it
		if candidate.LocationID == destination.LocationID {
			if i.debugLogging {
				log.Println("selecting: candidate = destination")
			}
			return candidate
		}

		// make sure we only consider candidates who are closer to the destination than we are
		if distanceToDestination >= currentToDestination {
			if i.debugLogging {
				log.Println("skipping: candidate wrong direction")
			}

			// this is critical
			// if we start moving the wrong direction and we are using express
			// shipping it's extremely likely that we have reached the closest
			// hub and thus all the candidates are farther away
			// for this last leg we need to use standard shipping
			if method == enum.Express {
				return i.NextLocation(current, destination, enum.Standard)
			}

			continue
		}

		// if we are express shipping then only consider hubs
		if method == enum.Express && candidate.Kind != enum.Hub {
			if i.debugLogging {
				log.Println("skipping: express shipping, candidate not a hub")
			}
			continue
		}

		// get the distance from our current location to the candidate
		currentToCandidate := planar.DistanceSquared(current.Position, candidate.Position)

		// only select destinations at least min distance away
		if currentToCandidate < minDistanceSquared {
			if i.debugLogging {
				log.Printf(
					"skipping: candidate(%0.2fkm) < minDistance(%0.2fkm)",
					math.Sqrt(currentToCandidate)/1000,
					math.Sqrt(minDistanceSquared)/1000,
				)
			}
			continue
		}

		// we found our match
		if i.debugLogging {
			log.Println("selecting: candidate")
		}
		return candidate
	}

	// if we fail to find the next nearest location - just send the package directly to the destination
	if i.debugLogging {
		log.Println("selecting: destination")
	}
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
