package workflow

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FakeClock is a Clock implementation for testing that allows manual time control.
// It is thread-safe and can be used with concurrent goroutines.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	timers  []*fakeTimer
	tickers []*fakeTicker
}

// NewFakeClock creates a new FakeClock starting at the given time
func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{
		now:     start,
		timers:  make([]*fakeTimer, 0),
		tickers: make([]*fakeTicker, 0),
	}
}

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) Since(t time.Time) time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now.Sub(t)
}

func (c *FakeClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	c.mu.Lock()
	defer c.mu.Unlock()

	deadline := c.now.Add(d)
	timer := &fakeTimer{
		clock:    c,
		deadline: deadline,
		ch:       ch,
		stopped:  false,
	}
	c.timers = append(c.timers, timer)

	return ch
}

func (c *FakeClock) NewTimer(d time.Duration) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()

	deadline := c.now.Add(d)
	timer := &fakeTimer{
		clock:    c,
		deadline: deadline,
		ch:       make(chan time.Time, 1),
		stopped:  false,
	}
	c.timers = append(c.timers, timer)

	return timer
}

func (c *FakeClock) NewTicker(d time.Duration) Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()

	ticker := &fakeTicker{
		clock:    c,
		interval: d,
		nextTick: c.now.Add(d),
		ch:       make(chan time.Time, 100),
		stopped:  false,
	}
	c.tickers = append(c.tickers, ticker)

	return ticker
}

// Advance moves the fake clock forward by the specified duration
// and fires any timers/tickers that should trigger during this period.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	targetTime := c.now.Add(d)

	for c.now.Before(targetTime) {
		nextEvent := targetTime

		for _, timer := range c.timers {
			timer.mu.Lock()
			if !timer.stopped && timer.deadline.After(c.now) && timer.deadline.Before(nextEvent) {
				nextEvent = timer.deadline
			}
			timer.mu.Unlock()
		}

		for _, ticker := range c.tickers {
			ticker.mu.Lock()
			if !ticker.stopped && ticker.nextTick.After(c.now) && ticker.nextTick.Before(nextEvent) {
				nextEvent = ticker.nextTick
			}
			ticker.mu.Unlock()
		}

		c.now = nextEvent

		for _, timer := range c.timers {
			timer.mu.Lock()
			if !timer.stopped && !timer.deadline.After(c.now) {
				select {
				case timer.ch <- c.now:
				default:
				}
				timer.stopped = true
			}
			timer.mu.Unlock()
		}

		for _, ticker := range c.tickers {
			ticker.mu.Lock()
			for !ticker.stopped && !ticker.nextTick.After(c.now) {
				select {
				case ticker.ch <- c.now:
				default:
				}
				ticker.nextTick = ticker.nextTick.Add(ticker.interval)
			}
			ticker.mu.Unlock()
		}

		if nextEvent.Equal(targetTime) {
			break
		}
	}
}

// fakeTimer implements Timer for testing
type fakeTimer struct {
	mu       sync.Mutex
	clock    *FakeClock
	deadline time.Time
	ch       chan time.Time
	stopped  bool
}

func (t *fakeTimer) C() <-chan time.Time {
	return t.ch
}

func (t *fakeTimer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stopped {
		return false
	}
	t.stopped = true
	return true
}

func (t *fakeTimer) Reset(d time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	wasActive := !t.stopped

	t.clock.mu.Lock()
	t.deadline = t.clock.now.Add(d)
	t.clock.mu.Unlock()

	t.stopped = false

	select {
	case <-t.ch:
	default:
	}

	return wasActive
}

// fakeTicker implements Ticker for testing
type fakeTicker struct {
	mu       sync.Mutex
	clock    *FakeClock
	interval time.Duration
	nextTick time.Time
	ch       chan time.Time
	stopped  bool
}

func (t *fakeTicker) C() <-chan time.Time {
	return t.ch
}

func (t *fakeTicker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
}

// Tests for real clock
func TestRealClock_Now(t *testing.T) {
	clock := NewRealClock()
	before := time.Now()
	got := clock.Now()
	after := time.Now()

	assert.True(t, !got.Before(before), "Now should not be before current time")
	assert.True(t, !got.After(after), "Now should not be after current time")
}

func TestRealClock_Since(t *testing.T) {
	clock := NewRealClock()
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	duration := clock.Since(start)

	assert.True(t, duration >= 10*time.Millisecond, "Since should return at least 10ms")
}

func TestRealClock_After(t *testing.T) {
	clock := NewRealClock()
	start := time.Now()
	ch := clock.After(10 * time.Millisecond)
	<-ch
	elapsed := time.Since(start)

	assert.True(t, elapsed >= 10*time.Millisecond, "After should fire after at least 10ms")
}

func TestRealClock_NewTimer(t *testing.T) {
	clock := NewRealClock()
	start := time.Now()
	timer := clock.NewTimer(10 * time.Millisecond)
	<-timer.C()
	elapsed := time.Since(start)

	assert.True(t, elapsed >= 10*time.Millisecond, "Timer should fire after at least 10ms")
}

func TestRealClock_NewTicker(t *testing.T) {
	clock := NewRealClock()
	ticker := clock.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	start := time.Now()
	<-ticker.C()
	elapsed := time.Since(start)

	assert.True(t, elapsed >= 10*time.Millisecond, "Ticker should fire after at least 10ms")
}

// Tests for fake clock
func TestFakeClock_Now(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(start)

	got := clock.Now()
	assert.Equal(t, start, got)
}

func TestFakeClock_Since(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(start)

	pastTime := start.Add(-1 * time.Hour)
	got := clock.Since(pastTime)
	assert.Equal(t, time.Hour, got)
}

func TestFakeClock_Advance_FiresTimersAtCorrectTime(t *testing.T) {
	tests := []struct {
		name          string
		timerDuration time.Duration
		advance       time.Duration
		shouldFire    bool
	}{
		{
			name:          "timer fires when advance reaches deadline",
			timerDuration: 5 * time.Second,
			advance:       5 * time.Second,
			shouldFire:    true,
		},
		{
			name:          "timer fires when advance exceeds deadline",
			timerDuration: 5 * time.Second,
			advance:       10 * time.Second,
			shouldFire:    true,
		},
		{
			name:          "timer does not fire when advance is before deadline",
			timerDuration: 5 * time.Second,
			advance:       3 * time.Second,
			shouldFire:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			clock := NewFakeClock(start)

			timer := clock.NewTimer(tt.timerDuration)
			clock.Advance(tt.advance)

			if tt.shouldFire {
				select {
				case got := <-timer.C():
					want := start.Add(tt.timerDuration)
					assert.Equal(t, want, got)
				default:
					t.Fatal("timer should have fired but did not")
				}
			} else {
				select {
				case <-timer.C():
					t.Fatal("timer should not have fired")
				default:
				}
			}
		})
	}
}

func TestFakeClock_Advance_FiresTickersMultipleTimes(t *testing.T) {
	tests := []struct {
		name            string
		tickerInterval  time.Duration
		advance         time.Duration
		expectedTickets int
	}{
		{
			name:            "ticker fires once for single interval",
			tickerInterval:  5 * time.Second,
			advance:         5 * time.Second,
			expectedTickets: 1,
		},
		{
			name:            "ticker fires multiple times for multiple intervals",
			tickerInterval:  5 * time.Second,
			advance:         15 * time.Second,
			expectedTickets: 3,
		},
		{
			name:            "ticker does not fire before first interval",
			tickerInterval:  5 * time.Second,
			advance:         3 * time.Second,
			expectedTickets: 0,
		},
		{
			name:            "ticker fires correct number for partial interval",
			tickerInterval:  5 * time.Second,
			advance:         12 * time.Second,
			expectedTickets: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			clock := NewFakeClock(start)

			ticker := clock.NewTicker(tt.tickerInterval)
			defer ticker.Stop()

			clock.Advance(tt.advance)

			tickCount := 0
			for {
				select {
				case <-ticker.C():
					tickCount++
				default:
					assert.Equal(t, tt.expectedTickets, tickCount)
					return
				}
			}
		})
	}
}

func TestFakeClock_StoppedTimersDoNotFire(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(start)

	timer := clock.NewTimer(5 * time.Second)
	wasActive := timer.Stop()

	assert.True(t, wasActive, "Stop should return true for active timer")

	clock.Advance(10 * time.Second)

	select {
	case <-timer.C():
		t.Fatal("stopped timer should not fire")
	default:
	}
}

func TestFakeClock_StoppedTickersDoNotFire(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(start)

	ticker := clock.NewTicker(5 * time.Second)
	ticker.Stop()

	clock.Advance(15 * time.Second)

	select {
	case <-ticker.C():
		t.Fatal("stopped ticker should not fire")
	default:
	}
}

func TestFakeClock_TimerReset(t *testing.T) {
	tests := []struct {
		name              string
		initialDuration   time.Duration
		stopBeforeReset   bool
		resetDuration     time.Duration
		advanceAfterReset time.Duration
		shouldFire        bool
	}{
		{
			name:              "reset active timer extends deadline",
			initialDuration:   5 * time.Second,
			stopBeforeReset:   false,
			resetDuration:     10 * time.Second,
			advanceAfterReset: 8 * time.Second,
			shouldFire:        false,
		},
		{
			name:              "reset active timer and advance to new deadline",
			initialDuration:   5 * time.Second,
			stopBeforeReset:   false,
			resetDuration:     10 * time.Second,
			advanceAfterReset: 10 * time.Second,
			shouldFire:        true,
		},
		{
			name:              "reset stopped timer reactivates it",
			initialDuration:   5 * time.Second,
			stopBeforeReset:   true,
			resetDuration:     3 * time.Second,
			advanceAfterReset: 3 * time.Second,
			shouldFire:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			clock := NewFakeClock(start)

			timer := clock.NewTimer(tt.initialDuration)

			if tt.stopBeforeReset {
				timer.Stop()
			}

			resetTime := clock.Now()
			wasActive := timer.Reset(tt.resetDuration)
			assert.Equal(t, !tt.stopBeforeReset, wasActive, "Reset should return whether timer was active")

			clock.Advance(tt.advanceAfterReset)

			if tt.shouldFire {
				select {
				case got := <-timer.C():
					want := resetTime.Add(tt.resetDuration)
					assert.Equal(t, want, got)
				default:
					t.Fatal("timer should have fired after reset")
				}
			} else {
				select {
				case <-timer.C():
					t.Fatal("timer should not have fired yet")
				default:
				}
			}
		})
	}
}

func TestFakeClock_ThreadSafety(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(start)

	var wg sync.WaitGroup
	iterations := 100

	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			clock.NewTimer(time.Duration(i) * time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			clock.NewTicker(time.Duration(i+1) * time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			clock.Advance(time.Millisecond)
		}
	}()

	wg.Wait()
}

func TestFakeClock_After(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(start)

	ch := clock.After(5 * time.Second)

	select {
	case <-ch:
		t.Fatal("After channel should not fire before advance")
	default:
	}

	clock.Advance(5 * time.Second)

	select {
	case got := <-ch:
		want := start.Add(5 * time.Second)
		assert.Equal(t, want, got)
	default:
		t.Fatal("After channel should fire after advance")
	}
}

func TestFakeClock_MultipleTimersFireInOrder(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(start)

	timer1 := clock.NewTimer(5 * time.Second)
	timer2 := clock.NewTimer(3 * time.Second)
	timer3 := clock.NewTimer(7 * time.Second)

	clock.Advance(10 * time.Second)

	t1 := <-timer1.C()
	t2 := <-timer2.C()
	t3 := <-timer3.C()

	assert.Equal(t, start.Add(5*time.Second), t1)
	assert.Equal(t, start.Add(3*time.Second), t2)
	assert.Equal(t, start.Add(7*time.Second), t3)
}

func TestTimer_StopReturnsFalseWhenAlreadyStopped(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(start)

	timer := clock.NewTimer(5 * time.Second)

	wasActive := timer.Stop()
	require.True(t, wasActive, "first Stop should return true")

	wasActive = timer.Stop()
	assert.False(t, wasActive, "second Stop should return false")
}
