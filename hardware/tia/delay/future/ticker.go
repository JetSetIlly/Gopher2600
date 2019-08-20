package future

import (
	"container/list"
	"strings"
)

// Ticker is used to group payloads for future triggering.
type Ticker struct {
	Label  string
	events list.List
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
