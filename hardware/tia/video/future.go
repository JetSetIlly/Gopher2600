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
}

// MachineInfo returns the ball sprite information in terse format
func (fut future) MachineInfo() string {
	if fut.remainingCycles == 0 {
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
	if fut.remainingCycles == 0 {
		return "no sch"
	}
	return fmt.Sprintf("%s(%d)", fut.label, fut.remainingCycles)
}

// schedule the pending future action
func (fut *future) schedule(cycles int, payload futurePayload, label string) {
	// silently preempt and forget about existing future events.
	// I'm pretty sure this is okay. the only time this can occur is during a
	// BRK instruction.
	//
	// there used to be a sanity panic here but the BRK
	// instruction would erroneoudly cause it to fail in certain instances. it
	// was easier to remove then to introduce special conditions.
	//
	// set remaining cycles:
	// + 1 because we trigger the payload on a count of 1 and use zero as the
	// off state
	// + 1 because we'll tick and consume a cycle immediately after scheduling
	fut.remainingCycles = cycles + 2

	fut.label = label
	fut.payload = payload
}

// isScheduled returns true if pending action has not yet resolved
func (fut future) isScheduled() bool {
	return fut.remainingCycles > 0
}

// tick moves the pending action counter on one step
func (fut *future) tick() bool {
	if fut.remainingCycles == 0 {
		return false
	}

	if fut.remainingCycles == 1 {
		fut.remainingCycles--
		return true
	}

	fut.remainingCycles--
	return false
}
