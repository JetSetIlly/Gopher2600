package future

// Instance represents a single future instance
type Instance struct {
	// label is a short decription describing the future payload
	label string

	// RemainingCycles is the number of remaining ticks before the pending
	// action is resolved
	RemainingCycles int

	// the value that is to be the result of the pending action
	payload func()
}

func (fut *Instance) schedule(cycles int, payload func(), label string) {
	// silently preempt and forget about existing future events.
	// I'm pretty sure this is okay. the only time this can occur is during a
	// BRK instruction.

	if fut.isScheduled() {
		panic("preempted future")
	}

	// there used to be a sanity panic here but the BRK
	// instruction would erroneoudly cause it to fail in certain instances. it
	// was easier to remove then to introduce special conditions.
	//
	// set remaining cycles:
	// + 1 because we trigger the payload on a count of 1 and use zero as the
	// off state
	// + 1 because we'll tick and consume a cycle immediately after scheduling
	fut.RemainingCycles = cycles + 2

	fut.label = label
	fut.payload = payload
}

func (fut Instance) isScheduled() bool {
	return fut.RemainingCycles > 0
}

func (fut *Instance) tick() bool {
	if fut.RemainingCycles == 0 {
		return false
	}

	if fut.RemainingCycles == 1 {
		fut.RemainingCycles--
		fut.payload()
		return true
	}

	fut.RemainingCycles--
	return false
}
