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

	// temporary cessation of ticks
	paused bool

	// completion of the event has been pushed back at least once
	pushed bool

	// the value that is to be the result of the pending action
	payload func()

	payloadWithArg func(interface{})
	payloadArg     interface{}
}

func (ev Event) String() string {
	label := strings.TrimSpace(ev.label)
	if label == "" {
		label = "[unlabelled event]"
	}
	return fmt.Sprintf("%s -> %d", label, ev.RemainingCycles)
}

func (ev *Event) isActive() bool {
	return ev.RemainingCycles >= 0
}

func (ev *Event) runPayload() {
	if ev.payloadWithArg != nil {
		ev.payloadWithArg(ev.payloadArg)
	} else {
		ev.payload()
	}
}

// Tick event forward one cycle
func (ev *Event) tick() bool {
	if !ev.isActive() {
		panic("events should not be ticked once they have expired under any circumstances")
	}

	if ev.paused {
		return false
	}

	ev.RemainingCycles--

	if ev.RemainingCycles == -1 {
		ev.runPayload()
		return true
	}

	return false
}

// Force can be used to immediately run the event's payload
//
// it is very important that the reference to the event is forgotten once
// Force() has been called
func (ev *Event) Force() {
	if !ev.isActive() {
		panic("cannot do that to a completed event")
	}

	ev.runPayload()
	ev.ticker.drop(ev)
	ev.RemainingCycles = -1
}

// Drop can be used to remove the event from the ticker queue without running
// the payload. Because the payload is not run then you should be careful to
// handle any cleanup that might otherwise occur (in the payload).
//
// it is very important that the reference to the event is forgotten once
// Drop() has been called
func (ev *Event) Drop() {
	if !ev.isActive() {
		panic("cannot do that to a completed event")
	}
	ev.ticker.drop(ev)
	ev.RemainingCycles = -1
}

// Push back event completion by effectively restarting the event. generally,
// an event will never need to be pushed back because it will have completed
// before an equivalent event is triggered. But sometimes, a second trigger
// will occur very quickly and it is more convenient to push, instead of
// droping and starting a new event.
func (ev *Event) Push() {
	if !ev.isActive() {
		panic("cannot do that to a completed event")
	}
	ev.RemainingCycles = ev.initialCycles
	ev.pushed = true
}

// Pause prevents the event from ticking any further until Resume or Restart is
// called
func (ev *Event) Pause() {
	if !ev.isActive() {
		panic("cannot do that to a completed event")
	}
	ev.paused = true
}

// JustStarted is true if no Tick()ing has taken place yet
func (ev Event) JustStarted() bool {
	if !ev.isActive() {
		panic("cannot do that to a completed event")
	}
	return ev.RemainingCycles == ev.initialCycles && !ev.pushed
}

// AboutToEnd is true if event resolves on next Tick()
// * optimisation: called a lot. pointer to Event to prevent duffcopy()
func (ev *Event) AboutToEnd() bool {
	if !ev.isActive() {
		panic("cannot do that to a completed event")
	}
	return ev.RemainingCycles == 0
}
