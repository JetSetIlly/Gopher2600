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

// Package Callfn facilitates the ARM CALLFN process common to both DPC+ and
// CDF* cartridge mappers. It does not handle the ARM itself and cartridge
// mappers that use it should take care in particular to Run() and Step() the
// ARM when required.
//
// It is not required when the ARM is intended to be run in "immediate" mode.
package callfn

import (
	"math"

	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// CallFn keeps track of the CallFn process common to both DPC+ and CDF*
// cartridge mappers.
type CallFn struct {
	// number of outstanding arm cycles
	remainingCycles float32

	// on ARM program conclusion we JMP to the address after the CALLFN
	ResumeAddr uint16

	// the number of remaining bytes to push during resume sequence
	resumeCount int

	// phantom reads happen all the time but we don't normally care about them.
	// but one place where it does matter is at the moment when a CALLFN is
	// finishing.
	//
	// the problem is caused by the conclusion of the ARM program not being
	// predictable. this means it's possible that it finishes sometime between
	// the placeholder NOP and the phantom read connected with that NOP (the
	// phantom read always happens).
	//
	// what we don't want to happen is to send the JMP opcode in response to
	// the phantom read. so, to prevent that we toggle the phantomOnResume
	// flag on every read during the period the ARM is executing. then, at the
	// precise moment the ARM processor is finishing (ie. armRemainingCyles <=
	// 0 AND resumeCount > 0) we can discard the first read event if we're
	// expecting a phantom read. for this to work correctly we should reset the
	// flag when beginning the CALLFN process.
	//
	// this feels like a special condition but it's really just information
	// about the state of the ARM that we aren't able to put anywhere else.
	phantomOnResume bool
}

// IsActive returns true if ARM program is still running.
func (cf *CallFn) IsActive() bool {
	return cf.remainingCycles > 0
}

const (
	jmpAbsolute = 0x4c
	nop         = 0xea
)

// Check state of CallFn. Returns true if it is active and false if it is not.
// If CallFn is active then the the value to put on the data bus is also
// returned. If CallFn is not active then the data bus value should be
// determined in the normal way (most probably by reading the cartridge ROM).
func (cf *CallFn) Check(addr uint16) (uint8, bool) {
	if cf.IsActive() {
		cf.phantomOnResume = !cf.phantomOnResume
		if cf.phantomOnResume {
			return nop, true
		}
		return 0x00, true
	}

	switch cf.resumeCount {
	case 4:
		// we always send at least one NOP instruction during a CALLFN. this
		// branch ensures that while playing nicely with the phantom read
		// problem
		cf.phantomOnResume = !cf.phantomOnResume
		if cf.phantomOnResume {
			return nop, true
		}
		cf.resumeCount--
		return 0x00, true
	case 3:
		cf.resumeCount--
		return jmpAbsolute, true
	case 2:
		cf.resumeCount--
		return uint8(cf.ResumeAddr & 0x00ff), true
	case 1:
		cf.resumeCount--
		return uint8(cf.ResumeAddr >> 8), true
	}

	// resume address after a CALLFN is the last read address (which will be
	// the address at which the CALLFN trigger was read) plus one. we also need
	// to add the cartridge bits because addresses are normalised by the
	// cartridge layer before passing to the mappers.
	//
	// the problem with this is that the cartridge mirror specified by the
	// address may be "wrong". not a problem from an execution point of view
	// but it might seem odd to someone monitoring closely in the debugger.
	cf.ResumeAddr = (addr | memorymap.OriginCartFxxxMirror) + 1

	return 0, false
}

// Accumulate cycles to the account for. Is safe to call if callfn is already
// active.
//
// There is no need to call this function if the ARM is operating in "immediate"
// mode. However, if you want to see the corrective JMP instruction at the end
// of a CALLFN then it is safe to call this function with cycles value of 0
func (cf *CallFn) Accumulate(cycles float32) {
	// if remaining cycles is zero then this is a new execution
	if cf.remainingCycles == 0 {
		cf.resumeCount = 4
		cf.phantomOnResume = false
	}

	// accumulate the new cycles
	cf.remainingCycles += cycles

	// we are no longer capping the number of cycles executed here. this is now
	// done entirely within in the arm7tdmi package.
	//
	// capping cycles here meant that the ARM program ran to completion, which
	// is not correct because that means the ARM memory is updated as though the
	// cap did not exist.
}

// Step forward one clock. Returns 0 or a value that should be used to call
// arm.Step()
//
// callfn.IsActive() should have been checked before calling this function. Also
// consider whether the function needs to be called at all - the ARM emulation
// might be in immediate mode
func (cf *CallFn) Step(vcsClock float32, armClock float32) float32 {
	// number of arm cycles consumed for every colour clock
	armCycles := float32(armClock / vcsClock)

	if cf.remainingCycles > armCycles {
		cf.remainingCycles -= armCycles
		return 0
	}

	remnantClock, _ := math.Modf(float64(armClock / (armCycles - cf.remainingCycles)))
	cf.remainingCycles = 0

	return float32(remnantClock)
}
