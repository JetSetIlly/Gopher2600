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

package lazyvalues

import (
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
)

// LazyCPU lazily accesses CPU information from the emulator.
type LazyCPU struct {
	val *LazyValues

	hasReset  atomic.Value // bool
	rdy       atomic.Value // bool
	pc        atomic.Value // registers.ProgramCounter
	a         atomic.Value // registers.Register
	x         atomic.Value // registers.Register
	y         atomic.Value // registers.Register
	sp        atomic.Value // registers.Register
	statusReg atomic.Value // registers.StatusRegister

	HasReset  bool
	RdyFlg    bool
	PC        registers.ProgramCounter
	A         registers.Register
	X         registers.Register
	Y         registers.Register
	SP        registers.Register
	StatusReg registers.StatusRegister
}

func newLazyCPU(val *LazyValues) *LazyCPU {
	return &LazyCPU{val: val}
}

func (lz *LazyCPU) push() {
	lz.hasReset.Store(lz.val.Dbg.VCS.CPU.HasReset())
	lz.rdy.Store(lz.val.Dbg.VCS.CPU.RdyFlg)
	lz.pc.Store(lz.val.Dbg.VCS.CPU.PC)
	lz.a.Store(lz.val.Dbg.VCS.CPU.A)
	lz.x.Store(lz.val.Dbg.VCS.CPU.X)
	lz.y.Store(lz.val.Dbg.VCS.CPU.Y)
	lz.sp.Store(lz.val.Dbg.VCS.CPU.SP)
	lz.statusReg.Store(lz.val.Dbg.VCS.CPU.Status)
}

func (lz *LazyCPU) update() {
	lz.HasReset, _ = lz.hasReset.Load().(bool)
	lz.RdyFlg, _ = lz.rdy.Load().(bool)
	lz.PC, _ = lz.pc.Load().(registers.ProgramCounter)
	lz.A, _ = lz.a.Load().(registers.Register)
	lz.X, _ = lz.x.Load().(registers.Register)
	lz.Y, _ = lz.y.Load().(registers.Register)
	lz.SP, _ = lz.sp.Load().(registers.Register)
	lz.StatusReg, _ = lz.statusReg.Load().(registers.StatusRegister)
}
