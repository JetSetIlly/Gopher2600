// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package rewind

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// SearchMemoryWrite runs an emulation between two states looking for the
// instance when the address is written to with the value (valueMask is applied
// to mask specific bits)
//
// The supplied target state is the upper limit of the search. The lower limit
// of the search is one frame before the target State.
//
// The supplied address will be normalised.
//
// Returns the most recent State at which the memory write was found. If a more
// recent address write is found but not the correct value, then no state is
// returned.
func (r *Rewind) SearchMemoryWrite(tgt *State, addr uint16, value uint8, valueMask uint8) (*State, error) {
	// matchingState is a snapshot of the the most recent search match
	var matchingState *State
	var mostRecentTVstate string

	// trace normalised address
	addr, _ = memorymap.MapAddress(addr, false)

	// create a new TV and VCS to search with
	searchTV, err := television.NewTelevision("NTSC")
	if err != nil {
		return nil, curated.Errorf("rewind: search: %v", err)
	}
	_ = searchTV.SetFPSCap(false)

	searchVCS, err := hardware.NewVCS(searchTV)
	if err != nil {
		return nil, curated.Errorf("rewind: search: %v", err)
	}

	// get current screen coordinates. the emulation will run until these
	// values are met, if not sooner.
	endCoords := tgt.TV.GetCoords()

	// find a recent state from the rewind history and plumb it our searchVCS
	idx, _, _ := r.findFrameIndex(endCoords.Frame)
	plumb(searchVCS, r.entries[idx])

	// loop until we reach (or just surpass) the target State
	done := false
	for !done && searchVCS.CPU.LastResult.Final {
		err = searchVCS.Step(nil)
		if err != nil {
			return nil, curated.Errorf("rewind: search: %v", err)
		}

		if searchVCS.Mem.LastAccessWrite && searchVCS.Mem.LastAccessAddressMapped == addr {
			if searchVCS.Mem.LastAccessValue&valueMask == value&valueMask {
				matchingState = snapshot(searchVCS, levelAdhoc)
			}
			mostRecentTVstate = searchTV.String()
		}

		// check to see if TV state exceeds the requested state
		searchCoords := searchVCS.TV.GetCoords()
		done = coords.GreaterThanOrEqual(searchCoords, endCoords)
	}

	// make sure the matching state is the last address match we found.
	if matchingState != nil && mostRecentTVstate != matchingState.TV.String() {
		matchingState = nil
	}

	return matchingState, nil
}

// SearchMemoryWrite runs an emulation between two states looking for the
// instance when the register is written to with the value (valueMask is
// applied to mask specific bits)
//
// The supplied target state is the upper limit of the search. The lower limit
// of the search is one frame before the target State.
//
// Returns the most recent State at which the register write was found. If a
// more recent register write is found but not the correct value, then no state
// is returned.
func (r *Rewind) SearchRegisterWrite(tgt *State, reg rune, value uint8, valueMask uint8) (*State, error) {
	// see commentary in SearchMemoryWrite(). although note that when
	// mostRecentTVSstate is noted is different in the case of
	// SearchRegisterWrite()
	var matchingState *State
	var mostRecentTVstate string

	searchTV, err := television.NewTelevision("NTSC")
	if err != nil {
		return nil, curated.Errorf("rewind: search: %v", err)
	}
	_ = searchTV.SetFPSCap(false)

	searchVCS, err := hardware.NewVCS(searchTV)
	if err != nil {
		return nil, curated.Errorf("rewind: search: %v", err)
	}

	// get current screen coordinates. the emulation will run until these
	// values are met, if not sooner.
	endCoords := tgt.TV.GetCoords()

	// find a recent state and plumb it into searchVCS
	idx, _, _ := r.findFrameIndex(endCoords.Frame)
	plumb(searchVCS, r.entries[idx])

	// onLoad() is called whenever a CPU register is loaded with a new value
	match := false
	onLoad := func(v uint8) {
		match = v&valueMask == value&valueMask

		// note TV state whenever register is loaded
		mostRecentTVstate = searchTV.String()
	}

	switch reg {
	case 'A':
		searchVCS.CPU.A.SetOnLoad(onLoad)
	case 'X':
		searchVCS.CPU.X.SetOnLoad(onLoad)
	case 'Y':
		searchVCS.CPU.Y.SetOnLoad(onLoad)
	default:
		panic(fmt.Sprintf("rewind: search: unrecognised CPU register (%c)", reg))
	}

	done := false
	for !done && searchVCS.CPU.LastResult.Final {
		err = searchVCS.Step(nil)
		if err != nil {
			return nil, curated.Errorf("rewind: search: %v", err)
		}

		// make snapshot of current state at CPU instruction boundary
		if match {
			match = false
			matchingState = snapshot(searchVCS, levelAdhoc)
		}

		// check to see if TV state exceeds the requested state
		searchCoords := searchVCS.TV.GetCoords()
		done = coords.GreaterThanOrEqual(searchCoords, endCoords)
	}

	// make sure the matching state is the last address match we found.
	if matchingState != nil && mostRecentTVstate != matchingState.TV.String() {
		matchingState = nil
	}

	return matchingState, nil
}
