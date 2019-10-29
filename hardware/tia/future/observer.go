package future

// Observer exposes only the function relating to the observing of events
type Observer interface {
	Observe(label string) (*Event, bool)
}

// Observe looks for the most recent event with the specified label. if it is
// found then it is returned along with the value true to indicate a match. if
// it is not found, then the most recent event (with whatever label) is
// returned along with false to indicate no match
func (tck Ticker) Observe(label string) (*Event, bool) {
	// start from the back of the active list. i.e the entry just in front of
	// the active sentinal
	e := tck.activeSentinal.Prev()
	for e != nil {
		v := e.Value.(*Event)

		// return match
		if v.label == label && v.RemainingCycles > -1 {
			return v, true
		}

		e = e.Prev()
	}

	// return most recent event if no match found
	e = tck.activeSentinal.Prev()
	if e == nil {
		return nil, false
	}

	return e.Value.(*Event), false
}
