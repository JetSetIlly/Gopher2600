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

	"github.com/jetsetilly/gopher2600/environment"
)

// TrackerEnvironment defines the subset of the Environment type required
// by a Tracker implementation
type TrackerEnvironment interface {
	IsEmulation(environment.Label) bool
}

// Tracker implementations display or otherwise record the state of the audio
// registers for each channel.
type Tracker interface {
	// AudioTick is called every video cycle
	AudioTick(env TrackerEnvironment, channel int, reg Registers)
}

// The TIA emulation takes two samples per scanline, so by definition the sample
// frequency is double the horizontal scan rate for the machine
//
// 31468.52 for NTSC
// 31250 for PAL
//
// an average of the two can be rounded to 31360
const AverageSampleFreq = 31360

// For a long time we used a sample frequency of 31400, or 15700*2
const OldSampleFreq = 31400

// The TIA audio volume state for both channles is sampled twice per scanline
const SamplesPerScanline = 2

// Audio is the implementation of the TIA audio sub-system
type Audio struct {
	env *environment.Environment

	// the reference frequency for all sound produced by the TIA is 30Khz.
	// this is the 3.58Mhz clock, which the TIA operates at, divided by
	// 114. that's one half of a scanline so we count to 228 and update
	// twice in that time
	clock228 int

	// the volume is sampled every colour clock and the volume at each clock is
	// summed. at fixed points, the volume is averaged
	sampleSum   []int
	sampleSumCt int

	// From the "Stella Programmer's Guide":
	//
	// "There are two audio circuits for generating sound. They are identical but
	// completely independent and can be operated simultaneously [...]"
	Channel0 channel
	Channel1 channel

	// the volume output for each channel
	Vol0 uint8
	Vol1 uint8

	// the addition of a tracker is not required
	tracker Tracker
}

// NewAudio is the preferred method of initialisation for the Audio sub-system.
func NewAudio(env *environment.Environment) *Audio {
	au := &Audio{
		env:       env,
		sampleSum: make([]int, 2),
	}

	return au
}

// Plumb audio into emulation
func (au *Audio) Plumb(env *environment.Environment) {
	au.env = env
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
	s.WriteString(au.Channel0.String())
	s.WriteString("  ch1: ")
	s.WriteString(au.Channel1.String())
	return s.String()
}

// UpdateTracker changes the state of the attached tracker. Should be called
// whenever any of the audio registers have changed.
func (au *Audio) UpdateTracker() {
}

// Step the audio on one TIA clock. The step will be filtered to produce a
// 30Khz clock.
func (au *Audio) Step() bool {
	if au.tracker != nil {
		// it's impossible for both channels to have changed in a single video cycle
		if au.Channel0.registersChanged {
			au.tracker.AudioTick(au.env, 0, au.Channel0.Registers)
			au.Channel0.registersChanged = false
		} else if au.Channel1.registersChanged {
			au.tracker.AudioTick(au.env, 1, au.Channel1.Registers)
			au.Channel1.registersChanged = false
		}
	}

	var changed bool

	// sum volume bits
	au.sampleSum[0] += int(au.Channel0.actualVolume())
	au.sampleSum[1] += int(au.Channel1.actualVolume())
	au.sampleSumCt++

	if (au.clock228 >= 8 && au.clock228 <= 10) || (au.clock228 >= 80 && au.clock228 <= 82) {
		au.Channel0.phase0()
		au.Channel1.phase0()
	}

	if (au.clock228 >= 36 && au.clock228 <= 38) || (au.clock228 >= 148 && au.clock228 <= 150) {
		au.Channel0.phase1()
		au.Channel1.phase1()

		// take average of sum of volume bits
		au.Vol0 = uint8(au.sampleSum[0]/au.sampleSumCt) & 0x0f
		au.Vol1 = uint8(au.sampleSum[1]/au.sampleSumCt) & 0x0f
		au.sampleSum[0] = 0
		au.sampleSum[1] = 0
		au.sampleSumCt = 0

		changed = true
	}

	// advance 228 clock and reset sample counter
	au.clock228 += 3
	if au.clock228 >= 228 {
		au.clock228 = 0
	}

	return changed
}
