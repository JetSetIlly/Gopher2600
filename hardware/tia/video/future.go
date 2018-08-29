package video

import "fmt"

type futurePayload interface{}

// future is a general purpose counter
type future struct {
	// label is a short decription describing the future payload
	label string

	active bool

	// remainingCycles is the number of remaining ticks before the pending
	// action is resolved
	remainingCycles int

	// the value that is to be the result of the pending action
	payload futurePayload

	// whether or not a scheduled operation has completed -- used primarily as
	// a sanity check
	unresolved bool
}

// MachineInfo returns the ball sprite information in terse format
func (fut future) MachineInfo() string {
	if !fut.unresolved {
		return "nothing scheduled"
	}
	suffix := ""
	if fut.remainingCycles != 1 {
		suffix = "s"
	}
	return fmt.Sprintf("%s in %d cycle%s", fut.label, fut.remainingCycles, suffix)
}

// MachineInfo returns the ball sprite information in verbose format
func (fut future) MachineInfoTerse() string {
	if !fut.unresolved {
		return "no sch"
	}
	return fmt.Sprintf("%s(%d)", fut.label, fut.remainingCycles)
}

// schedule the pending future action
func (fut *future) schedule(cycles int, payload futurePayload, label string) {
	if fut.unresolved {
		panic(fmt.Sprintf("scheduling future (%s) before previous operation (%s) is resolved", label, fut.label))
	}

	// remaining cycles
	// + 1 because we'll tick and consume a cycle immediately after scheduling
	fut.remainingCycles = cycles + 1

	fut.label = label
	fut.payload = payload
	fut.unresolved = true
}

// isScheduled returns true if pending action has not yet resolved
func (fut future) isScheduled() bool {
	return !fut.unresolved
}

// tick moves the pending action counter on one step
func (fut *future) tick() bool {
	if fut.unresolved {
		if fut.remainingCycles == 0 {
			fut.unresolved = false
			return true
		}

		fut.remainingCycles--
	}

	return false
}
