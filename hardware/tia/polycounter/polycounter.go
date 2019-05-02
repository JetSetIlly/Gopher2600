package polycounter

import (
	"fmt"
)

// Polycounter implements the VCS method of counting. It is doesn't require
// special initialisation so is a good candidate for struct embedding
type Polycounter struct {
	table polycounterTable

	// this implementation of the VCS polycounter uses a regular go-integer
	Count int

	// the phase ranges from 0 and MaxPhase
	Phase int

	// reset point is the value of count at which the polycounter should reset
	// to 0
	ResetPoint int
}

// MachineInfoTerse returns the polycounter information in terse format
func (pk Polycounter) MachineInfoTerse() string {
	return fmt.Sprintf("%d@%d", pk.Count, pk.Phase)
}

// MachineInfo returns the polycounter information in verbose format
func (pk Polycounter) MachineInfo() string {
	return fmt.Sprintf("(%d) %s@%d", pk.Count, pk.table[pk.Count], pk.Phase)
}

// map String to MachineInfo
func (pk Polycounter) String() string {
	return pk.MachineInfo()
}

// SetResetPattern specifies the pattern at which the polycounter automatically
// resets during a Tick(). this should be called at least once or the reset
// pattern will be "000000" which is probably not what you want
func (pk *Polycounter) SetResetPattern(resetPattern string) {
	pk.ResetPoint = pk.table.LookupPattern(resetPattern)
}

// SetResetPoint specifies the point at which the polycounter automatically
func (pk *Polycounter) SetResetPoint(resetPoint int) {
	if resetPoint > len(pk.table) {
		panic(fmt.Sprintf("cannot set reset point to %d for a polycounter table of length %d", resetPoint, len(pk.table)))
	}
	pk.ResetPoint = resetPoint
}

// ResetPhase resets the phase *only*
func (pk *Polycounter) ResetPhase() {
	pk.Phase = 0
}

// Reset leaves the polycounter in its "zero" state. resetPattern
func (pk *Polycounter) Reset() {
	pk.Count = 0
	pk.Phase = 0
}

// Sync is used to synchronise two polycounters
// -- positive offsets adjusts the reset point to the right
// -- used for moving missile position to the player position
func (pk *Polycounter) Sync(pko *Polycounter, offset int) {
	if pk.ResetPoint != pko.ResetPoint {
		panic(fmt.Sprintf("cannot Sync() two polycounters with different reset points"))
	}

	pk.Count = pko.Count
	pk.Phase = pko.Phase

	// moving to the right means subtracting from the polycounter count/phase
	pk.Count -= offset / (MaxPhase + 1)
	if pk.Count > pk.ResetPoint {
		pk.Count = pk.Count - pk.ResetPoint
	} else if pk.Count < 0 {
		pk.Count = pk.ResetPoint
	}
	pk.Phase -= offset % (MaxPhase + 1)
	if pk.Phase > MaxPhase {
		pk.Phase = pk.Phase - MaxPhase
	} else if pk.Phase < 0 {
		pk.Phase = (MaxPhase + 1) + pk.Phase
		pk.Count--
		if pk.Count < 0 {
			pk.Count = pk.ResetPoint
		}
	}
}

// Tick advances the count to the next state - returns true if counter has
// looped. the force argument allows the function to be called and for the loop
// to definitely take place. we use this in the VCS during for the RSYNC check
func (pk *Polycounter) Tick() bool {
	if pk.CycleOnNextTick() {
		pk.Reset()
		return true
	}

	pk.Phase++
	if pk.Phase > MaxPhase {
		pk.ResetPhase()
		pk.Count++
	}

	return false
}

// Match check whether polycounter is at the given count, any phase
func (pk Polycounter) Match(count int) bool {
	return pk.Count == count
}

// MatchPhase checks whether polycounter is at the given count and given phase
func (pk Polycounter) MatchPhase(count, phase int) bool {
	return pk.Count == count && pk.Phase == phase
}

// MatchEnd checks whether polycounter is at the *end* (ie. last phase) of the given count
func (pk Polycounter) MatchEnd(count int) bool {
	return pk.Count == count && pk.Phase == MaxPhase
}

// MatchBeginning checks whether polycounter is at the *beginning* (ie. first phase) of the given count
func (pk Polycounter) MatchBeginning(count int) bool {
	return pk.Count == count && pk.Phase == 0
}

// Pixel returns the color clock when expressed as a pixel
func (pk Polycounter) Pixel() int {
	return (pk.Count * 4) + pk.Phase - 68
}

// CycleOnNextTick checks to see if polycounter is about to cycle
func (pk Polycounter) CycleOnNextTick() bool {
	return pk.Count == pk.ResetPoint && pk.Phase == MaxPhase
}

// CycledOnLastTick checks to see if polycounter has just cycled
func (pk Polycounter) CycledOnLastTick() bool {
	return pk.Count == 0 && pk.Phase == 0
}
