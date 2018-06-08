package video

type futurePayload interface{}

// future is a general purpose counter
type future struct {
	// remainingCycles is the number of remaining ticks before the pending
	// action is resolved
	remainingCycles int

	// the value that is to be the result of the pending action
	payload futurePayload
}

// newFuture is the preferred method of initialisation for the pending type
func newFuture() *future {
	dc := new(future)
	if dc == nil {
		return nil
	}
	dc.remainingCycles = -1
	dc.payload = true
	return dc
}

// schedule the pending future action
func (dc *future) schedule(cycles int, payload futurePayload) {
	dc.remainingCycles = cycles
	dc.payload = payload
}

// isScheduled returns true if pending action has not yet resolved
func (dc future) isScheduled() bool {
	return dc.remainingCycles > -1
}

// tick moves the pending action counter on one step
func (dc *future) tick() bool {
	if dc.remainingCycles == 0 {
		dc.remainingCycles--
		return true
	}

	if dc.isScheduled() {
		dc.remainingCycles--
	}

	return false
}
