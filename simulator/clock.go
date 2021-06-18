package simulator

import (
	"time"
)

// Clock keeps track of the simulator's current time
type Clock struct {
	now time.Time
}

func NewClock(start time.Time) *Clock {
	return &Clock{
		now: start,
	}
}

// Tick the clock and return the amount of time that has passed
func (c *Clock) Tick(interval time.Duration) time.Time {
	c.now = c.now.Add(interval)
	return c.now
}

// Set the clock to a specific time
func (c *Clock) Set(t time.Time) {
	c.now = t
}

func (c *Clock) Now() time.Time {
	return c.now
}
