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
	"github.com/jetsetilly/gopher2600/logger"
)

// Gatherer should be run (with the Check() function) every video cycle.
type Gatherer struct {
	vcs      *hardware.VCS
	renderer Renderer

	// history of gathered reflections
	history []ReflectedVideoStep

	// the next index to be used by Step()
	historyIdx int

	// state of emulation
	emulationState emulation.State
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

// OnInstructionEnd should be called at the conclusion of a single CPU
// instruction execution. This should be called appropriately by the execution
// loop, in addition to OnVideoCycle(), because not all information about the
// CPU can be gathered accurately at the time of the previous video step. And
// by the time of the next video step the information will be lost.
func (ref *Gatherer) OnInstructionEnd(bank mapper.BankInfo) error {
	// in practical terms, we need this function to make sure that the CPU
	// field is up-to-date. the other fields won't have changed

	// update previous entry. if the history is empty then a new entry will be
	// added, rather than updated. this can cause problems with reflection
	// renderers if processing of history is strictly sequential. for this
	// reason there is an advisory comment in the Renderer interface
	// definition.
	if ref.historyIdx > 0 {
		ref.historyIdx--
	}

	return ref.OnVideoCycle(bank)
}

// OnVideoCycle should be called every video cycle to record the current state
// of the system. See also OnInstructionEnd() in order to gather a complete
// reflection of the system over time.
func (ref *Gatherer) OnVideoCycle(bank mapper.BankInfo) error {
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

	// reflector is not paused so we record VideoStep for later processing.
	ref.history[ref.historyIdx] = v
	ref.historyIdx++

	if ref.historyIdx >= len(ref.history) {
		return ref.render()
	}

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

	// reset reflection history
	ref.historyIdx = 0

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
	// if new state is not paused then render history - the pause state is
	// handled in the Step() function
	if ref.renderer != nil {
		return ref.render()
	}

	return nil
}
