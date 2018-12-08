package future

import "fmt"

// Group is used to buffer payloads for future triggering.
type Group struct {
	id      int
	singles [3]Instance
}

// MachineInfo returns the ball sprite information in terse format
func (fut Group) MachineInfo() string {
	sng := fut.singles[fut.id]

	if sng.RemainingCycles == 0 {
		return "nothing scheduled"
	}
	suffix := ""
	if sng.RemainingCycles != 1 {
		suffix = "s"
	}
	return fmt.Sprintf("%s in %d cycle%s", sng.label, sng.RemainingCycles, suffix)
}

// MachineInfoTerse returns the ball sprite information in verbose format
func (fut Group) MachineInfoTerse() string {
	sng := fut.singles[fut.id]

	if sng.RemainingCycles == 0 {
		return "no sch"
	}
	return fmt.Sprintf("%s(%d)", sng.label, sng.RemainingCycles)
}

// Schedule the pending future action
func (fut *Group) Schedule(cycles int, payload func(), label string) *Instance {
	fut.id++
	if fut.id >= len(fut.singles) {
		fut.id = 0
	}

	fut.singles[fut.id].schedule(cycles, payload, label)

	return &fut.singles[fut.id]
}

// IsScheduled returns true if pending action has not yet resolved
func (fut Group) IsScheduled() bool {
	return fut.singles[0].isScheduled() || fut.singles[1].isScheduled() || fut.singles[2].isScheduled()
}

// Tick moves the pending action counter on one step
func (fut *Group) Tick() bool {
	return fut.singles[0].tick() || fut.singles[1].tick() || fut.singles[2].tick()
}
