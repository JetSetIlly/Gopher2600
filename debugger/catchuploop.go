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

package debugger

import (
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// CatchUpLoop will run the emulation from it's current state until the specificied
// frame/scanline/horizpos has been reached.
//
// If the breakpoint can't be reached, because the specified TV state is in the
// past, then false is returned.
func (dbg *Debugger) CatchUpLoop(frame int, scanline int, horizpos int) bool {
	nf := dbg.VCS.TV.GetState(signal.ReqFramenum)
	ny := dbg.VCS.TV.GetState(signal.ReqScanline)
	nx := dbg.VCS.TV.GetState(signal.ReqHorizPos)

	if nf > frame || (nf == frame && ny > scanline) || (nf == frame && ny == scanline && nx >= horizpos) {
		return false
	}

	dbg.lastBank = dbg.VCS.Mem.Cart.GetBank(dbg.VCS.CPU.PC.Address())

	for !(nf > frame || (nf == frame && ny > scanline) || (nf == frame && ny == scanline && nx >= horizpos)) {
		err := dbg.VCS.Step(func() error {
			return dbg.reflect.Check(dbg.lastBank)
		})
		if err != nil {
			return false
		}

		dbg.lastBank = dbg.VCS.Mem.Cart.GetBank(dbg.VCS.CPU.PC.Address())

		nf = dbg.VCS.TV.GetState(signal.ReqFramenum)
		ny = dbg.VCS.TV.GetState(signal.ReqScanline)
		nx = dbg.VCS.TV.GetState(signal.ReqHorizPos)
	}

	return true
}
