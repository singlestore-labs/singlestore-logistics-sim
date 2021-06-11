package simulator

import (
	"time"
)

// Clock keeps track of the simulator's current time
type Clock struct {
	now      time.Time
	interval time.Duration
}

func NewClock(start time.Time, interval time.Duration) *Clock {
	return &Clock{
		now:      start,
		interval: interval,
	}
}

// Tick the clock and return the amount of time that has passed
func (c *Clock) Tick() time.Duration {
	c.now = c.now.Add(c.interval)
	return c.interval
}

func (c *Clock) Now() time.Time {
	return c.now
}
