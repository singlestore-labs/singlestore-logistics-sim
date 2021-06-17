package simulator

import (
	"container/heap"
	"fmt"
	"log"
	"math"
	"math/rand"
	"simulator/enum"
	"sort"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
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
	Population  int
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
		Position:   orb.Point{dbloc.Longitude, dbloc.Latitude},
		Population: dbloc.Population,
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
	qt            *quadtree.Quadtree
	ht            map[int64]*Location
	popSorted     []*Location
	minPopulation int
	maxPopulation int

	debugLogging bool
}

func NewLocationIndexFromDB(dblocs []DBLocation, debugLogging bool) (*LocationIndex, error) {
	idx := &LocationIndex{
		qt:           quadtree.New(orb.Bound{Min: minGeoPoint, Max: maxGeoPoint}),
		ht:           make(map[int64]*Location),
		popSorted:    make([]*Location, 0, len(dblocs)),
		debugLogging: debugLogging,
	}
	for _, dbloc := range dblocs {
		loc := NewLocationFromDB(dbloc)
		err := idx.qt.Add(loc)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		idx.ht[loc.LocationID] = loc
		idx.popSorted = append(idx.popSorted, loc)
	}

	// popSorted must be sorted by population
	sort.Slice(idx.popSorted, func(i, j int) bool {
		return idx.popSorted[i].Population < idx.popSorted[j].Population
	})
	idx.minPopulation = idx.popSorted[0].Population
	idx.maxPopulation = idx.popSorted[len(idx.popSorted)-1].Population

	const (
		Pi      = math.Pi
		PiOver2 = Pi / 2
	)

	locationFilter := func(origin *Location, direction string, kind enum.LocationKind) func(p orb.Pointer) bool {
		return func(p orb.Pointer) bool {
			loc := p.(*Location)
			if loc == origin {
				return false
			}
			if kind != enum.Any && kind != loc.Kind {
				return false
			}

			angle := geo.Bearing(origin.Position, loc.Position)

			switch direction {
			case "ne":
				return angle > 0 && angle <= 90
			case "nw":
				return angle > 90 && angle <= 180
			case "sw":
				return angle > -180 && angle <= -90
			case "se":
				return angle > -90 && angle <= 0
			}

			panic(fmt.Sprintf("unknown direction: '%s'", direction))
		}
	}

	// precompute and store knearest for each location per direction (nw, ne, sw, se)
	nearestPerDirection := 2
	directions := []string{"ne", "nw", "sw", "se"}
	buf := make([]orb.Pointer, 0, nearestPerDirection)

	start := time.Now()
	log.Printf("generating knearest location index, this can take awhile...")
	for _, loc := range idx.popSorted {
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

	log.Printf("finished generating knearest location index in %s", time.Since(start))

	return idx, nil
}

func (idx *LocationIndex) NextLocation(current *Location, destination *Location, method enum.DeliveryMethod) *Location {
	// our current squared distance to the destination
	currentToDestination := geo.Distance(current.Position, destination.Position)

	// min distance should be at least 1/2 of the remaining distance to the destination
	// or 200km, whichever is larger
	minDistance := math.Max(currentToDestination / 2, 200*1000)

	seen := make(map[*Location]struct{})
	seen[current] = empty

	q := NewLocationQueue()
	q.PushLocation(current, 0, currentToDestination)

	if idx.debugLogging {
		log.Printf("NextLocation: current to destination = %0.2fkm", currentToDestination/1000)
	}

	considered := 0
	if idx.debugLogging {
		defer func() {
			log.Printf("NextLocation considered %d candidates, remaining queue size: %d", considered, q.Len())
		}()
	}

	for q.Len() > 0 {
		candidate, distanceToDestination := q.PopLocation()
		considered++

		nearest := candidate.Nearest
		if method == enum.Express {
			nearest = candidate.NearestHubs
		}

		// prepend unseen neighbors based on distance to destination
		for _, neighbor := range nearest {
			if _, ok := seen[neighbor]; !ok {
				seen[neighbor] = empty

				// get the distance from the neighbor location to our destination
				neighborToDestination := geo.Distance(neighbor.Position, destination.Position)

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

		if idx.debugLogging {
			log.Printf("NextLocation: candidate (distance to destination = %0.2fkm)", distanceToDestination/1000)
		}

		// if one of the candidates is our destination, select it
		if candidate.LocationID == destination.LocationID {
			if idx.debugLogging {
				log.Println("selecting: candidate = destination")
			}
			return candidate
		}

		// make sure we only consider candidates who are closer to the destination than we are
		if distanceToDestination >= currentToDestination {
			if idx.debugLogging {
				log.Println("skipping: candidate wrong direction")
			}

			// this is critical
			// if we start moving the wrong direction and we are using express
			// shipping it's extremely likely that we have reached the closest
			// hub and thus all the candidates are farther away
			// for this last leg we need to use standard shipping
			if method == enum.Express {
				return idx.NextLocation(current, destination, enum.Standard)
			}

			continue
		}

		// if we are express shipping then only consider hubs
		if method == enum.Express && candidate.Kind != enum.Hub {
			if idx.debugLogging {
				log.Println("skipping: express shipping, candidate not a hub")
			}
			continue
		}

		// get the distance from our current location to the candidate
		currentToCandidate := geo.Distance(current.Position, candidate.Position)

		// only select destinations at least min distance away
		if currentToCandidate < minDistance {
			if idx.debugLogging {
				log.Printf(
					"skipping: distance(%0.2fkm) < minDistance(%0.2fkm)",
					currentToCandidate/1000,
					minDistance/1000,
				)
			}
			continue
		}

		// we found our match
		if idx.debugLogging {
			log.Printf("selecting: candidate (%0.2fkm from current)", currentToCandidate/1000)
		}
		return candidate
	}

	// if we fail to find the next nearest location - just send the package directly to the destination
	if idx.debugLogging {
		log.Println("selecting: destination")
	}
	return destination
}

func (idx *LocationIndex) Lookup(locationID int64) (*Location, error) {
	if l, ok := idx.ht[locationID]; ok {
		return l, nil
	}
	return nil, errors.Errorf("location %d not found", locationID)
}

func randBetween(min int, max int) int {
	return rand.Intn(max-min+1) + min
}

// Rand returns a random location in the index for which the filter returns true
// filter can be nil which implies no filter
func (idx *LocationIndex) Rand(filter quadtree.FilterFunc) *Location {
	randPopulation := randBetween(idx.minPopulation, idx.maxPopulation)

	i := sort.Search(len(idx.popSorted), func(i int) bool {
		return idx.popSorted[i].Population >= randPopulation
	})

	attempts := 10
	for attempt := 0; attempt < attempts; attempt++ {
		candidate := idx.popSorted[i]
		if filter == nil || filter(candidate) {
			return candidate
		}
		i = (i + 1) % len(idx.popSorted)
	}

	log.Fatalf("failed to find candidate in %d attempts", attempts)
	return nil
}
