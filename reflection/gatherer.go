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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/logger"
)

// Gatherer should be run (with the Check() function) every video cycle.
type Gatherer struct {
	vcs      *hardware.VCS
	renderer Renderer

	// history of gathered reflections
	history []ReflectedVideoStep

	// state of emulation
	emulationState emulation.State

	// the index of the most recent signal returned by television.GetLastSignal()
	lastIdx int
}

// NewGatherer is the preferred method of initialisation for the Monitor type.
func NewGatherer(vcs *hardware.VCS) *Gatherer {
	return &Gatherer{
		vcs:     vcs,
		history: make([]ReflectedVideoStep, television.MaxSignalHistory),
	}
}

// AddRenderer adds an implementation of the Renderer interface to the Reflector.
func (ref *Gatherer) AddRenderer(renderer Renderer) {
	ref.renderer = renderer
}

// Step should be called every video cycle to record a complete
// reflection of the system.
func (ref *Gatherer) Step(bank mapper.BankInfo) error {
	v := ReflectedVideoStep{
		CPU:               ref.vcs.CPU.LastResult,
		WSYNC:             !ref.vcs.CPU.RdyFlg,
		Bank:              bank,
		VideoElement:      ref.vcs.TIA.Video.LastElement,
		Signal:            ref.vcs.TV.GetLastSignal(),
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

	idx := int((v.Signal & signal.Index) >> signal.IndexShift)
	ref.history[idx] = v

	// nullify entries at the head of the array that do not have a
	// corresponding signal. this works because signals returned by
	// GetLastSignal should be linear.
	//
	// unlike the nullify loop in NewFrame(), this does not have a
	// corresponding constructin in the television implementation.
	for i := ref.lastIdx; i < idx-1; i++ {
		ref.history[i] = ReflectedVideoStep{}
	}

	ref.lastIdx = idx

	return nil
}

// push history to reflection renderer
func (ref *Gatherer) render() error {
	if ref.emulationState != emulation.Rewinding {
		if ref.renderer != nil {
			if err := ref.renderer.Reflect(ref.history); err != nil {
				return curated.Errorf("reflection: %v", err)
			}
		}
	}

	return nil
}

// SetEmulationState is called by emulation whenever state changes. How we
// handle reflections depends on the current state.
func (ref *Gatherer) SetEmulationState(state emulation.State) {
	prev := ref.emulationState
	ref.emulationState = state

	switch prev {
	case emulation.Rewinding:
		ref.render()
	}

	switch state {
	case emulation.Paused:
		err := ref.render()
		if err != nil {
			logger.Logf("reflection", "%v", err)
		}
	}
}

// NewFrame implements the television.FrameTrigger interface.
func (ref *Gatherer) NewFrame(_ television.FrameInfo) error {
	// nullify unused entries at end of frame
	//
	// note that this echoes a similar construct in the television.NewFrame()
	// function. it's important that this happens here or the results in the
	// Renderer will not be satisfactory.
	for i := ref.lastIdx; i < len(ref.history); i++ {
		ref.history[i] = ReflectedVideoStep{}
	}
	ref.lastIdx = 0

	// if new state is not paused then render history - the pause state is
	// handled in the Step() function
	return ref.render()
}
