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

	v := tck.pool.Remove(tck.pool.Front()).(*Event)
	tck.events.PushBack(v)

	v.label = label
	v.initialCycles = delay
	v.RemainingCycles = delay
	v.paused = false
	v.pushed = false
	v.payload = payload

	// no need to update the pointer to the Ticker instance

	return v
}
