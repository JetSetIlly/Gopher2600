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
	var matchingState *State

	addr, _ = memorymap.MapAddress(addr, true)

	rewindTV, err := television.NewTelevision("NTSC")
	if err != nil {
		return nil, curated.Errorf("rewind: search: %v", err)
	}
	_ = rewindTV.SetFPSCap(false)

	rewindVCS, err := hardware.NewVCS(rewindTV)
	if err != nil {
		return nil, curated.Errorf("rewind: search: %v", err)
	}

	// get current screen coordinates. the emulation will run until these
	// values are met, if not sooner.
	ef := tgt.TV.GetState(signal.ReqFramenum)
	es := tgt.TV.GetState(signal.ReqScanline)
	ec := tgt.TV.GetState(signal.ReqClock)

	// find a recent state and plumb it into rewindVCS
	idx, _, _ := r.findFrameIndex(ef - 1)
	plumb(rewindVCS, r.entries[idx])

	done := false
	for !done && rewindVCS.CPU.LastResult.Final {
		err = rewindVCS.Step(nil)
		if err != nil {
			return nil, curated.Errorf("rewind: search: %v", err)
		}

		f := rewindVCS.TV.GetState(signal.ReqFramenum)
		s := rewindVCS.TV.GetState(signal.ReqScanline)
		c := rewindVCS.TV.GetState(signal.ReqClock)

		if rewindVCS.Mem.LastAccessWrite && rewindVCS.Mem.LastAccessAddressMapped == addr && rewindVCS.Mem.LastAccessValue&valueMask == value&valueMask {
			matchingState = snapshot(rewindVCS, levelAdhoc)
		}

		// check to see if TV state exceeds the requested state
		done = f > ef || (f == ef && s > es) || (f == ef && s == es && c >= ec)
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
	var matchingState *State

	rewindTV, err := television.NewTelevision("NTSC")
	if err != nil {
		return nil, curated.Errorf("rewind: search: %v", err)
	}
	_ = rewindTV.SetFPSCap(false)

	rewindVCS, err := hardware.NewVCS(rewindTV)
	if err != nil {
		return nil, curated.Errorf("rewind: search: %v", err)
	}

	// get current screen coordinates. the emulation will run until these
	// values are met, if not sooner.
	ef := tgt.TV.GetState(signal.ReqFramenum)
	es := tgt.TV.GetState(signal.ReqScanline)
	ec := tgt.TV.GetState(signal.ReqClock)

	// find a recent state and plumb it into rewindVCS
	idx, _, _ := r.findFrameIndex(ef - 1)
	plumb(rewindVCS, r.entries[idx])

	onLoad := func(val uint8) {
		if val&valueMask == value&valueMask {
			matchingState = snapshot(rewindVCS, levelAdhoc)
		}
	}

	switch reg {
	case "A":
		rewindVCS.CPU.A.SetOnLoad(onLoad)
	case "X":
		rewindVCS.CPU.X.SetOnLoad(onLoad)
	case "Y":
		rewindVCS.CPU.Y.SetOnLoad(onLoad)
	}

	done := false
	for !done && rewindVCS.CPU.LastResult.Final {
		err = rewindVCS.Step(nil)
		if err != nil {
			return nil, curated.Errorf("rewind: search: %v", err)
		}

		f := rewindVCS.TV.GetState(signal.ReqFramenum)
		s := rewindVCS.TV.GetState(signal.ReqScanline)
		c := rewindVCS.TV.GetState(signal.ReqClock)

		// check to see if TV state exceeds the requested state
		done = f > ef || (f == ef && s > es) || (f == ef && s == es && c >= ec)
	}

	return matchingState, nil
}
