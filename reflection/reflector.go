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
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
)

// Reflector should be run (with the Check() function) every video cycle.
type Reflector struct {
	vcs            *hardware.VCS
	renderer       Renderer
	emulationState govern.State

	history []ReflectedVideoStep

	// the index of the most recent signal returned by television.GetLastSignal()
	lastIdx int
}

// NewReflector is the preferred method of initialisation for the Monitor type.
func NewReflector(vcs *hardware.VCS) *Reflector {
	ref := &Reflector{vcs: vcs}
	ref.Clear()
	return ref
}

// AddRenderer adds an implementation of the Renderer interface to the Reflector.
func (ref *Reflector) AddRenderer(renderer Renderer) {
	ref.renderer = renderer
}

// Clear existing reflected information.
func (ref *Reflector) Clear() {
	ref.history = make([]ReflectedVideoStep, specification.AbsoluteMaxClks)
}

// SetEmulationState is called by emulation whenever state changes. How we
// handle reflections depends on the current state.
func (ref *Reflector) SetEmulationState(state govern.State) {
	prev := ref.emulationState
	ref.emulationState = state

	switch prev {
	case govern.Rewinding:
		ref.render()
	}

	switch state {
	case govern.Paused:
		err := ref.render()
		if err != nil {
			logger.Logf("reflection", "%v", err)
		}
	}
}

// Step should be called every video cycle to record a complete
// reflection of the system.
func (ref *Reflector) Step(bank mapper.BankInfo) error {
	sig := ref.vcs.TV.GetLastSignal()

	// check that signal is not the NoSignal signal
	//
	// at the time of writng, this can sometimes happen if the VCS has been
	// reset but the emulator loop has not been unwound. The newly reset TV
	// will return an invalid signal leading to an index that is too large
	if sig == signal.NoSignal {
		return nil
	}

	idx := int((sig & signal.Index) >> signal.IndexShift)
	h := ref.history[idx : idx+1]

	h[0].CPU = ref.vcs.CPU.LastResult
	h[0].WSYNC = !ref.vcs.CPU.RdyFlg
	h[0].Bank = bank
	h[0].VideoElement = ref.vcs.TIA.Video.LastElement
	h[0].Signal = sig
	h[0].Collision = *ref.vcs.TIA.Video.Collisions
	h[0].IsHblank = ref.vcs.TIA.Hblank
	h[0].CoProcState = ref.vcs.Mem.Cart.CoProcState()

	if ref.vcs.TIA.Hmove.Future.IsActive() {
		h[0].Hmove.Delay = true
		h[0].Hmove.DelayCt = ref.vcs.TIA.Hmove.Future.Remaining()
	}
	if ref.vcs.TIA.Hmove.Latch {
		h[0].Hmove.Latch = true
		h[0].Hmove.RippleCt = ref.vcs.TIA.Hmove.Ripple
	}

	h[0].RSYNCalign, h[0].RSYNCreset = ref.vcs.TIA.RSYNCstate()

	// nullify entries at the head of the array that do not have a
	// corresponding signal. we do this because the first index of a signal
	// after a NewFrame might be different that the previous frame
	for i := ref.lastIdx; i < idx-1; i++ {
		ref.history[i] = ReflectedVideoStep{}
	}
	ref.lastIdx = idx

	return nil
}

// push history to reflection renderer
func (ref *Reflector) render() error {
	if ref.emulationState != govern.Rewinding {
		if ref.renderer != nil {
			if err := ref.renderer.Reflect(ref.history); err != nil {
				return curated.Errorf("reflection: %v", err)
			}
		}
	}

	return nil
}

// NewFrame implements the television.FrameTrigger interface.
func (ref *Reflector) NewFrame(_ television.FrameInfo) error {
	// nullify unused entries at end of frame
	//
	// note that this echoes a similar construct in the television.NewFrame()
	// function. it's important that this happens here or the results in the
	// Renderer will not be satisfactory.
	for i := ref.lastIdx; i < len(ref.history); i++ {
		ref.history[i] = ReflectedVideoStep{}
	}
	ref.lastIdx = 0

	return ref.render()
}
