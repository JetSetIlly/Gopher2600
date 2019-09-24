package future

import (
	"container/list"
	"strings"
)

// Ticker is used to group payloads for future triggering.
type Ticker struct {
	Label  string
	events list.List
	pool   list.List
}

// NewTicker is the only method of initialisation for the Ticker type
func NewTicker(label string) *Ticker {
	tck := &Ticker{Label: label}

	for i := 0; i < 6; i++ {
		tck.pool.PushBack(&Event{ticker: tck, RemainingCycles: -1})
	}

	return tck
}

// MachineInfo returns future ticker information in verbose format
func (tck Ticker) MachineInfo() string {
	s := strings.Builder{}
	for e := tck.events.Front(); e != nil; e = e.Next() {
		if tck.Label != "" {
			s.WriteString(tck.Label)
			s.WriteString(": ")
		}
		s.WriteString(e.Value.(*Event).String())
		s.WriteString("\n")
	}
	return s.String()
}

// MachineInfoTerse returns future ticker information in terse format
func (tck Ticker) MachineInfoTerse() string {
	e := tck.events.Back()
	if e == nil {
		return ""
	}

	s := strings.Builder{}

	if tck.Label != "" {
		s.WriteString(tck.Label)
		s.WriteString(": ")
	}

	// terse return just the first event in the list
	s.WriteString(e.Value.(*Event).String())
	if e.Next() != nil {
		s.WriteString(" [+]")
	}

	return s.String()
}

// Tick moves the pending action counter on one step
func (tck *Ticker) Tick() bool {
	r := false

	e := tck.events.Front()
	for e != nil {
		if e.Value.(*Event).tick() {
			r = true
			n := e.Next()
			v := tck.events.Remove(e)
			tck.pool.PushBack(v)
			e = n
		} else {
			e = e.Next()
		}
	}

	return r
}

func (tck *Ticker) drop(ev *Event) {
	e := tck.events.Front()
	for e != nil {
		if ev == e.Value.(*Event) {
			v := tck.events.Remove(e)
			tck.pool.PushBack(v)
			return
		}
		e = e.Next()
	}

	panic("cannot drop an event that is not in the list of active events")
}
