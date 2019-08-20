package future

// Scheduler exposes only the functions relating to scheduling of events
type Scheduler interface {
	Schedule(cycles int, payload func(), label string) *Event
	IsScheduled() bool
}

// Schedule the pending future action
func (tck *Ticker) Schedule(delay int, payload func(), label string) *Event {
	if delay <= 0 {
		payload()
		return nil
	}

	ins := &Event{ticker: tck, label: label, InitialCycles: delay, RemainingCycles: delay, payload: payload}
	tck.events.PushBack(ins)

	return ins
}

// IsScheduled returns true if there are any outstanding future events
func (tck Ticker) IsScheduled() bool {
	return tck.events.Len() > 0
}
