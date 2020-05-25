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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package reflection

import (
	"github.com/jetsetilly/gopher2600/hardware"
)

// Monitor watches for writes to specific video related memory locations. when
// these locations are written to, a signal is sent to the Renderer
// implementation. moreover, if the monitor detects that the effect of the
// memory write is delayed or sustained, then the signal is repeated as
// appropriate.
type Monitor struct {
	vcs      *hardware.VCS
	renderer Renderer
}

// NewMonitor is the preferred method of initialisation for the Monitor type
func NewMonitor(vcs *hardware.VCS, renderer Renderer) *Monitor {
	mon := &Monitor{
		vcs:      vcs,
		renderer: renderer,
	}

	return mon
}

// Check should be called every video cycle to record the current state of the
// emulation/system
func (mon *Monitor) Check() error {
	res := LastResult{
		CPU:          mon.vcs.CPU.LastResult,
		WSYNC:        !mon.vcs.CPU.RdyFlg,
		Bank:         mon.vcs.Mem.Cart.GetBank(mon.vcs.CPU.LastResult.Address),
		VideoElement: mon.vcs.TIA.LastVideoElement,
		TV:           mon.vcs.TV.GetLastSignal(),
	}

	if err := mon.renderer.Reflect(res); err != nil {
		return nil
	}

	return nil
}
