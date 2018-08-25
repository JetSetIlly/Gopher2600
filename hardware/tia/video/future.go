package video

import "fmt"

type futurePayload interface{}

// future is a general purpose counter
type future struct {
	// label is a short decription describing the future payload
	label string

	// remainingCycles is the number of remaining ticks before the pending
	// action is resolved
	remainingCycles int

	// the value that is to be the result of the pending action
	payload futurePayload

	// whether or not a scheduled operation has completed -- used primarily as
	// a sanity check
	unresolved bool
}

// schedule the pending future action
func (fut *future) schedule(cycles int, payload futurePayload, label string) {
	if fut.unresolved {
		panic(fmt.Sprintf("scheduling future (%s) before previous operation (%s) is resolved", label, fut.label))
	}
	fut.label = label
	fut.remainingCycles = cycles + 1
	fut.payload = payload
	fut.unresolved = true
}

// isScheduled returns true if pending action has not yet resolved
func (fut future) isScheduled() bool {
	return fut.remainingCycles > 0
}

// tick moves the pending action counter on one step
func (fut *future) tick() bool {
	if fut.remainingCycles == 1 {
		fut.remainingCycles--
		fut.unresolved = false
		return true
	}

	if fut.remainingCycles > 0 {
		fut.remainingCycles--
	}

	return false
}
