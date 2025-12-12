package workflow

import "time"

// Clock abstracts time operations to enable testability.
// Production code uses realClock which delegates to the standard time package.
// Tests use FakeClock which allows controlling time advancement.
type Clock interface {
	// Now returns the current time
	Now() time.Time
	// Since returns the duration since t
	Since(t time.Time) time.Duration
	// After returns a channel that fires after duration d
	After(d time.Duration) <-chan time.Time
	// NewTimer creates a new Timer that will send the current time on its channel after duration d
	NewTimer(d time.Duration) Timer
	// NewTicker creates a new Ticker that will send the current time on its channel at intervals of duration d
	NewTicker(d time.Duration) Ticker
}

// Timer represents a single event timer
type Timer interface {
	// C returns the timer's time channel
	C() <-chan time.Time
	// Stop prevents the Timer from firing
	Stop() bool
	// Reset changes the timer to expire after duration d
	Reset(d time.Duration) bool
}

// Ticker represents a repeating timer
type Ticker interface {
	// C returns the ticker's time channel
	C() <-chan time.Time
	// Stop turns off the ticker
	Stop()
}

// realClock implements Clock using the standard time package
type realClock struct{}

// NewRealClock creates a new Clock that uses the standard time package
func NewRealClock() Clock {
	return &realClock{}
}

func (c *realClock) Now() time.Time {
	return time.Now()
}

func (c *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (c *realClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (c *realClock) NewTimer(d time.Duration) Timer {
	return &realTimer{timer: time.NewTimer(d)}
}

func (c *realClock) NewTicker(d time.Duration) Ticker {
	return &realTicker{ticker: time.NewTicker(d)}
}

// realTimer wraps time.Timer to implement the Timer interface
type realTimer struct {
	timer *time.Timer
}

func (t *realTimer) C() <-chan time.Time {
	return t.timer.C
}

func (t *realTimer) Stop() bool {
	return t.timer.Stop()
}

func (t *realTimer) Reset(d time.Duration) bool {
	return t.timer.Reset(d)
}

// realTicker wraps time.Ticker to implement the Ticker interface
type realTicker struct {
	ticker *time.Ticker
}

func (t *realTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t *realTicker) Stop() {
	t.ticker.Stop()
}
