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

package audio

import (
	"strings"
)

// Tracker implementations display or otherwise record the state of the audio
// registers for each channel.
type Tracker interface {
	// Tick is called every video cycle
	Tick(channel int, reg Registers)
}

// SampleFreq represents the number of samples generated per second. This is
// the 30Khz reference frequency desribed in the Stella Programmer's Guide.
const SampleFreq = 31400

// Audio is the implementation of the TIA audio sub-system, using Ron Fries'
// method. Reference source code here:
//
// https://raw.githubusercontent.com/alekmaul/stella/master/emucore/TIASound.c
type Audio struct {
	// the reference frequency for all sound produced by the TIA is 30Khz.
	// this is the 3.58Mhz clock, which the TIA operates at, divided by
	// 114. that's one half of a scanline so we count to 228 and update
	// twice in that time
	clock228 int

	// From the "Stella Programmer's Guide":
	//
	// "There are two audio circuits for generating sound. They are identical but
	// completely independent and can be operated simultaneously [...]"
	channel0 channel
	channel1 channel

	// the volume output for each channel
	Vol0 uint8
	Vol1 uint8

	tracker Tracker
}

// NewAudio is the preferred method of initialisation for the Audio sub-system.
func NewAudio() *Audio {
	return &Audio{}
}

func (au *Audio) Reset() {
	au.clock228 = 0
	au.channel0 = channel{}
	au.channel1 = channel{}
	au.Vol0 = 0
	au.Vol1 = 0
}

// SetTracker adds a Tracker implementation to the Audio sub-system.
func (au *Audio) SetTracker(tracker Tracker) {
	au.tracker = tracker
}

// Snapshot creates a copy of the TIA Audio sub-system in its current state.
func (au *Audio) Snapshot() *Audio {
	n := *au
	return &n
}

func (au *Audio) String() string {
	s := strings.Builder{}
	s.WriteString("ch0: ")
	s.WriteString(au.channel0.String())
	s.WriteString("  ch1: ")
	s.WriteString(au.channel1.String())
	return s.String()
}

// UpdateTracker changes the state of the attached tracker. Should be called
// whenever any of the audio registers have changed.
func (au *Audio) UpdateTracker() {
	if au.tracker == nil {
		return
	}

	// it's impossible for both channels to have changed in a single video cycle
	if au.channel0.registersChanged {
		au.tracker.Tick(0, au.channel0.registers)
	} else if au.channel1.registersChanged {
		au.tracker.Tick(1, au.channel1.registers)
	}
	au.channel0.registersChanged = false
	au.channel1.registersChanged = false
}

// Step the audio on one TIA clock. The step will be filtered to produce a
// 30Khz clock.
func (au *Audio) Step() bool {
	au.clock228++
	if au.clock228 >= 228 {
		au.clock228 = 0
		return false
	}

	switch au.clock228 {
	case 10:
		au.channel0.phase0()
		au.channel1.phase0()
		return false
	case 82:
		au.channel0.phase0()
		au.channel1.phase0()
		return false
	case 38:
		au.channel0.phase1()
		au.channel1.phase1()
	case 150:
		au.channel0.phase1()
		au.channel1.phase1()
	default:
		return false
	}

	au.Vol0 = au.channel0.actualVol
	au.Vol1 = au.channel1.actualVol

	return true
}
