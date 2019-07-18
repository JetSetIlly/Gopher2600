package future

import (
	"container/list"
	"strings"
)

// Ticker is used to group payloads for future triggering.
type Ticker struct {
	events list.List
}

// MachineInfo returns future ticker information in verbose format
func (tck Ticker) MachineInfo() string {
	s := strings.Builder{}
	for e := tck.events.Front(); e != nil; e = e.Next() {
		s.WriteString(e.Value.(*Event).String())
		s.WriteString("\n")
	}
	return s.String()
}

// MachineInfoTerse returns future ticker information in terse format
func (tck Ticker) MachineInfoTerse() string {
	e := tck.events.Front()
	if e == nil {
		return ""
	}

	s := strings.Builder{}

	// terse return just the first event in the list
	s.WriteString(e.Value.(*Event).String())
	if e.Next() != nil {
		s.WriteString(" [+]")
	}

	return s.String()
}

// Schedule the pending future action
func (tck *Ticker) Schedule(cycles int, payload func(), label string) *Event {
	ins := &Event{ticker: tck, label: label, initialCycles: cycles, RemainingCycles: cycles, payload: payload}
	tck.events.PushBack(ins)
	return ins
}

// IsScheduled returns true if there are any outstanding future events
func (tck Ticker) IsScheduled() bool {
	return tck.events.Len() > 0
}

// Tick moves the pending action counter on one step
func (tck *Ticker) Tick() bool {
	r := false

	e := tck.events.Front()
	for e != nil {
		t := e.Value.(*Event).Tick()
		r = r || t

		if t {
			f := e.Next()
			tck.events.Remove(e)
			e = f
		} else {
			e = e.Next()
		}
	}

	return r
}

// Drop can be used to immediately run the future payload
func (tck *Ticker) Drop(ins *Event) {
	e := tck.events.Front()
	for e != nil {
		i := e.Value.(*Event)
		if i == ins {
			tck.events.Remove(e)
			break
		} else {
			e = e.Next()
		}
	}
}
