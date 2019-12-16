package future

// Scheduler exposes only the functions relating to scheduling of events
type Scheduler interface {
	Schedule(delay int, payload func(), label string) *Event
	ScheduleWithArg(delay int, payload func(arg interface{}), arg interface{}, label string) *Event
}

func (tck *Ticker) schedule(delay int, label string) *Event {
	// take element from the back of the pool (the inactive half)
	e := tck.pool.Back()
	v := e.Value.(*Event)

	// sanity check to make sure the active and inactive lists have not collided
	// this should never happen. if it does then poolSize is too small
	if e == tck.activeSentinal || v.isActive() {
		// if we ever get to this point then the data being run is probably not
		// a valid VCS ROM. returning nil is nonsensical for normal operation
		// but that's okay because we're reasonably sure we're in a nonsensical
		// situation anyway
		return nil
	}

	// move to the back of the active list (in front of the active sentinal)
	tck.pool.MoveBefore(e, tck.activeSentinal)

	// update event information
	v.label = label
	v.initialCycles = delay
	v.remainingCycles = delay
	v.paused = false
	v.pushed = false

	return v
}

// Schedule the pending future action
func (tck *Ticker) Schedule(delay int, payload func(), label string) *Event {
	if delay < 0 {
		payload()
		return nil
	}

	v := tck.schedule(delay, label)
	if v == nil {
		return nil
	}
	v.payload = payload
	v.payloadWithArg = nil
	v.payloadArg = nil

	return v
}

// ScheduleWithArg schedules the pending future action with an argument to the
// payload function
func (tck *Ticker) ScheduleWithArg(delay int, payload func(interface{}), arg interface{}, label string) *Event {
	if delay < 0 {
		payload(arg)
		return nil
	}

	v := tck.schedule(delay, label)
	v.payload = nil
	v.payloadWithArg = payload
	v.payloadArg = arg

	return v
}
