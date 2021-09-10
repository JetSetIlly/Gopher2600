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

package reflection

import (
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

// Gatherer should be run (with the Check() function) every video cycle.
type Gatherer struct {
	vcs      *hardware.VCS
	renderer Renderer

	// whether reflection is enabled or not
	enabled bool

	// whether the reflector is paused or not. this affects how the VideoStep
	// history is handled.
	paused bool

	// VideoStep history. used to buffer reflected VideoSteps when the
	// Reflector is unpaused.
	history []ReflectedVideoStep

	// start/end of the history buffer
	s, e int
}

// NewGatherer is the preferred method of initialisation for the Monitor type.
func NewGatherer(vcs *hardware.VCS) *Gatherer {
	return &Gatherer{
		vcs:     vcs,
		history: make([]ReflectedVideoStep, television.MaxSignalHistory),
		enabled: true,
	}
}

// AddRenderer adds an implementation of the Renderer interface to the Reflector.
func (ref *Gatherer) AddRenderer(renderer Renderer) {
	ref.renderer = renderer
}

// EnableReflection implements the Reflector interface.
func (ref *Gatherer) EnableReflection(enabled bool) {
	ref.enabled = enabled
}

// Step should be called every video cycle to record the current state of the
// emulation/system.
func (ref *Gatherer) Step(bank mapper.BankInfo) error {
	if !ref.enabled {
		return nil
	}

	v := ReflectedVideoStep{
		CPU:               ref.vcs.CPU.LastResult,
		WSYNC:             !ref.vcs.CPU.RdyFlg,
		Bank:              bank,
		VideoElement:      ref.vcs.TIA.Video.LastElement,
		TV:                ref.vcs.TV.GetLastSignal(),
		Collision:         *ref.vcs.TIA.Video.Collisions,
		IsHblank:          ref.vcs.TIA.Hblank,
		CoprocessorActive: bank.ExecutingCoprocessor,
	}

	if ref.vcs.TIA.Hmove.Future.IsActive() {
		v.Hmove.Delay = true
		v.Hmove.DelayCt = ref.vcs.TIA.Hmove.Future.Remaining()
	}
	if ref.vcs.TIA.Hmove.Latch {
		v.Hmove.Latch = true
		v.Hmove.RippleCt = ref.vcs.TIA.Hmove.Ripple
	}

	v.RSYNCalign, v.RSYNCreset = ref.vcs.TIA.RSYNCstate()

	// if reflector is paused then we need to reflect the pixel now
	if ref.renderer != nil && ref.paused {
		ref.renderer.UpdatingPixels(true)
		defer ref.renderer.UpdatingPixels(false)
		if err := ref.renderer.Reflect(v); err != nil {
			return err
		}
		return nil
	}

	// reflector is not paused so we record VideoStep for later processing.
	if ref.e < television.MaxSignalHistory {
		ref.history[ref.e] = v
		ref.e++
	}

	return nil
}

// Pause implements the television.PauseTrigger interface.
func (ref *Gatherer) Pause(pause bool) error {
	ref.paused = pause

	// process all outstanding VideoStep in the history on pause
	if ref.renderer != nil && ref.paused {
		ref.renderer.UpdatingPixels(true)
		defer ref.renderer.UpdatingPixels(false)
		for i := ref.s; i < ref.e; i++ {
			if err := ref.renderer.Reflect(ref.history[i]); err != nil {
				return err
			}
		}
		ref.s = ref.e
	}

	return nil
}

// NewFrame implements the television.FrameTrigger interface.
func (ref *Gatherer) NewFrame(_ television.FrameInfo) error {
	// if reflector is not paused then we must process all outstanding
	// VideoStep in the history
	if ref.renderer != nil && !ref.paused {
		ref.renderer.UpdatingPixels(true)
		defer ref.renderer.UpdatingPixels(false)
		for i := ref.s; i < ref.e; i++ {
			if err := ref.renderer.Reflect(ref.history[i]); err != nil {
				return err
			}
		}
	}

	// reset history indexes
	ref.s = 0
	ref.e = 0

	return nil
}
