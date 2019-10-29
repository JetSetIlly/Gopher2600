package future

// Scheduler exposes only the functions relating to scheduling of events
type Scheduler interface {
	Schedule(delay int, payload func(), label string) *Event
}

// Schedule the pending future action
func (tck *Ticker) Schedule(delay int, payload func(), label string) *Event {
	if delay < 0 {
		payload()
		return nil
	}

	// take element from the back of the pool (the inactive half)
	e := tck.pool.Back()
	v := e.Value.(*Event)

	// sanity check to make sure the active and inactive lists have not collided
	// this should never happen. if it does then poolSize is too small
	if e == tck.activeSentinal || v.isActive() {
		panic("pool of future events has been exhausted")
	}

	// move to the back of the active list (in front of the active sentinal)
	tck.pool.MoveBefore(e, tck.activeSentinal)

	// update event information
	v.label = label
	v.initialCycles = delay
	v.RemainingCycles = delay
	v.paused = false
	v.pushed = false
	v.payload = payload

	return v
}
