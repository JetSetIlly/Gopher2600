package future

import (
	"fmt"
	"strings"
)

// Event represents a single occurence (contained in payload) that is to be
// deployed in the future
type Event struct {
	// the future ticker this event belongs to
	ticker *Ticker

	// label is a short decription describing the future payload
	label string

	// the number of cycles the event began with
	initialCycles int

	// the number of remaining ticks before the pending action is resolved
	RemainingCycles int

	paused    bool
	completed bool

	// the value that is to be the result of the pending action
	payload func()
}

func (ev Event) String() string {
	label := strings.TrimSpace(ev.label)
	if label == "" {
		label = "[unlabelled event]"
	}
	return fmt.Sprintf("%s -> %d", label, ev.RemainingCycles)
}

// Tick event forward one cycle
func (ev *Event) Tick() bool {
	if ev.paused {
		return false
	}

	// 0 is the trigger state
	if ev.RemainingCycles == 0 {
		ev.RemainingCycles--
		ev.payload()
		ev.completed = true
		return true
	}

	// -1 is the off state
	if ev.RemainingCycles != -1 {
		ev.RemainingCycles--
	}

	return false
}

// Force can be used to immediately run the event's payload
func (ev *Event) Force() {
	ev.payload()
	ev.ticker.Drop(ev)
	ev.completed = true
}

// Drop can be used to remove the event from the ticker queue without running
// the payload. Because the payload is not run then you should be careful to
// handle any cleanup that might otherwise occur (in the payload).
func (ev *Event) Drop() {
	ev.ticker.Drop(ev)
	ev.completed = true
}

// Pause prevents the event from ticking any further until Resume or Restart is
// called
func (ev *Event) Pause() {
	ev.paused = true
}

// Resume a previously paused event
func (ev *Event) Resume() {
	ev.paused = false
}

// Restart an event
func (ev *Event) Restart() {
	ev.RemainingCycles = ev.initialCycles
	ev.paused = false
}

// Completed indicates whether the events has run it's course
func (ev Event) Completed() bool {
	return ev.completed
}
