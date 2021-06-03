package simulator

import (
	"sync"
	"time"
)

// Clock keeps track of the simulator's current time
// it is thread safe
type Clock struct {
	now      time.Time
	interval time.Duration
	mu       sync.RWMutex
}

func NewClock(start time.Time, interval time.Duration) *Clock {
	return &Clock{
		now:      start,
		interval: interval,
	}
}

// Tick the clock and return the amount of time that has passed
func (c *Clock) Tick() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(c.interval)
	return c.interval
}

func (c *Clock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}
