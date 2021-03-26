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
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// SearchMemoryWrite runs an emulation from the supplied state until the point
// when the address has been written to with the supplied value.
//
// The supplied address should be normalised for this function to work
// correctly.
//
// If a write to the address has been found then valueMask is applied to the
// value being written to see if it matches the supplied value. The supplied
// value will also be masked before comparison.
//
// Returns the State at which the memory write was found.
func (r *Rewind) SearchMemoryWrite(tgt *State, addr uint16, value uint8, valueMask uint8) (*State, error) {
	// we'll match every instruction that writes value to the addr (applying
	// valueMask). we'll also not the TV state whenever addr matches,
	// regardless of value. this way we can detect whether the matching state
	// is actually the most recent match.
	//
	// we only want to return a matchingState if it's the most recent addr/value match.
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
	ef := tgt.TV.GetState(signal.ReqFramenum)
	es := tgt.TV.GetState(signal.ReqScanline)
	ec := tgt.TV.GetState(signal.ReqClock)

	// find a recent state from the rewind history and plumb it our searchVCS
	idx, _, _ := r.findFrameIndex(ef)
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
		sf := searchVCS.TV.GetState(signal.ReqFramenum)
		ss := searchVCS.TV.GetState(signal.ReqScanline)
		sc := searchVCS.TV.GetState(signal.ReqClock)
		done = sf > ef || (sf == ef && ss > es) || (sf == ef && ss == es && sc >= ec)
	}

	// make sure the matching state is the last address match we found.
	if matchingState != nil && mostRecentTVstate != matchingState.TV.String() {
		return nil, curated.Errorf("rewind: false match at %04x", addr)
	}

	return matchingState, nil
}

// SearchRegisterWrite runs an emulation from the supplied state until the point
// when the register has been written to with the supplied value.
//
// If a write to the register has been found then valueMask is applied to the
// value being written to see if it matches the supplied value. The supplied
// value will also be masked before comparison.
//
// Returns the State at which the memory write was found.
func (r *Rewind) SearchRegisterWrite(tgt *State, reg string, value uint8, valueMask uint8) (*State, error) {
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
	ef := tgt.TV.GetState(signal.ReqFramenum)
	es := tgt.TV.GetState(signal.ReqScanline)
	ec := tgt.TV.GetState(signal.ReqClock)

	// find a recent state and plumb it into searchVCS
	idx, _, _ := r.findFrameIndex(ef)
	plumb(searchVCS, r.entries[idx])

	// onLoad() is called whenever a CPU register is loaded with a new value
	match := false
	onLoad := func(val uint8) {
		match = val&valueMask == value&valueMask

		// note TV state whenever register is loaded
		mostRecentTVstate = searchTV.String()
	}

	switch reg {
	case "A":
		searchVCS.CPU.A.SetOnLoad(onLoad)
	case "X":
		searchVCS.CPU.X.SetOnLoad(onLoad)
	case "Y":
		searchVCS.CPU.Y.SetOnLoad(onLoad)
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
		sf := searchVCS.TV.GetState(signal.ReqFramenum)
		ss := searchVCS.TV.GetState(signal.ReqScanline)
		sc := searchVCS.TV.GetState(signal.ReqClock)
		done = sf > ef || (sf == ef && ss > es) || (sf == ef && ss == es && sc >= ec)
	}

	// make sure the matching state is the last address match we found.
	if matchingState != nil && mostRecentTVstate != matchingState.TV.String() {
		return nil, curated.Errorf("rewind: false match in %s", reg)
	}

	return matchingState, nil
}
