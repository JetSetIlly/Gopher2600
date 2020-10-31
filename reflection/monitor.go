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

// Monitor should be run (with the Check() function) every video cycle. The
// (reflection) Renderer's Reflect() function is consequently also called every
// video cycle with a populated instance of LastResult.
type Monitor struct {
	vcs        *hardware.VCS
	renderer   Renderer
	history    [television.MaxSignalHistory]Reflection
	historyIdx int
}

// NewMonitor is the preferred method of initialisation for the Monitor type.
func NewMonitor(vcs *hardware.VCS, renderer Renderer) *Monitor {
	return &Monitor{
		vcs:      vcs,
		renderer: renderer,
	}
}

// Check should be called every video cycle to record the current state of the
// emulation/system.
func (mon *Monitor) Check(bank mapper.BankInfo) error {
	res := Reflection{
		CPU:          mon.vcs.CPU.LastResult,
		WSYNC:        !mon.vcs.CPU.RdyFlg,
		Bank:         bank,
		VideoElement: mon.vcs.TIA.Video.LastElement,
		TV:           mon.vcs.TV.GetLastSignal(),
		Hblank:       mon.vcs.TIA.Hblank,
		Collision:    mon.vcs.TIA.Video.Collisions.Activity.String(),
		Unchanged:    mon.vcs.TIA.Video.Unchanged,
	}

	// reflect HMOVE state
	if mon.vcs.TIA.FutureHmove.IsActive() {
		res.Hmove.Delay = true
		res.Hmove.DelayCt = mon.vcs.TIA.FutureHmove.Remaining()
	}
	if mon.vcs.TIA.HmoveLatch {
		res.Hmove.Latch = true
		res.Hmove.RippleCt = mon.vcs.TIA.HmoveCt
	}

	if mon.historyIdx < television.MaxSignalHistory {
		mon.history[mon.historyIdx] = res
		mon.historyIdx++
	}

	return nil
}

// SyncReflectionPixel implements the television.ReflectionSynchronising interface.
func (mon *Monitor) SyncReflectionPixel(idx int) error {
	if err := mon.renderer.Reflect(mon.history[idx]); err != nil {
		return err
	}
	mon.historyIdx = idx
	return nil
}

// SyncFrame implements the television.ReflectionSynchronising interface.
func (mon *Monitor) SyncFrame() {
	mon.historyIdx = 0
}
