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

package lazyvalues

import (
	"sync"
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
)

// LazyCPU lazily accesses CPU information from the emulator.
type LazyCPU struct {
	val *Values

	atomicHasReset   atomic.Value //bool
	atomicRdy        atomic.Value // bool
	atomicPCAddr     atomic.Value // uint16
	atomicLastResult atomic.Value // execution.Result
	atomicStatusReg  atomic.Value // registers.StatusRegister

	HasReset bool
	RdyFlg   bool

	// PCaddr is a numeric value rather than a string representation as
	// can be found when requesting a value from RegisterString()
	PCaddr uint16

	LastResult execution.Result
	StatusReg  registers.StatusRegister

	// register labels/value require a generic register. note use of mutex for
	// map access
	atomicRegLabelsMux   sync.RWMutex
	atomicRegValuesMux   sync.RWMutex
	atomicRegBitwidthMux sync.RWMutex

	atomicRegLabels   map[registers.Generic]atomic.Value // string
	atomicRegValues   map[registers.Generic]atomic.Value // string
	atomicRegBitwidth map[registers.Generic]atomic.Value // int
}

func newLazyCPU(val *Values) *LazyCPU {
	lz := &LazyCPU{val: val}
	lz.atomicRegLabels = make(map[registers.Generic]atomic.Value)
	lz.atomicRegValues = make(map[registers.Generic]atomic.Value)
	lz.atomicRegBitwidth = make(map[registers.Generic]atomic.Value)
	return lz
}

func (lz *LazyCPU) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicHasReset.Store(lz.val.VCS.CPU.HasReset())
		lz.atomicRdy.Store(lz.val.VCS.CPU.RdyFlg)
		lz.atomicPCAddr.Store(lz.val.VCS.CPU.PC.Address())
		lz.atomicLastResult.Store(lz.val.VCS.CPU.LastResult)
		lz.atomicStatusReg.Store(*lz.val.VCS.CPU.Status)
	})
	lz.HasReset, _ = lz.atomicHasReset.Load().(bool)
	lz.RdyFlg, _ = lz.atomicRdy.Load().(bool)
	lz.PCaddr, _ = lz.atomicPCAddr.Load().(uint16)
	lz.LastResult, _ = lz.atomicLastResult.Load().(execution.Result)
	lz.StatusReg, _ = lz.atomicStatusReg.Load().(registers.StatusRegister)
}

// RegLabel returns the label for the queried register
func (lz *LazyCPU) RegLabel(reg registers.Generic) string {
	if lz.val.Dbg == nil {
		return ""
	}

	// label of register will neved change to if value is in atomicRegLabels
	// map we don't need to read it again

	lz.atomicRegLabelsMux.RLock()
	defer lz.atomicRegLabelsMux.RUnlock()
	if v, ok := lz.atomicRegLabels[reg]; ok {
		return v.Load().(string)
	}

	lz.val.Dbg.PushRawEvent(func() {
		var a atomic.Value
		a.Store(reg.Label())
		lz.atomicRegLabelsMux.Lock()
		lz.atomicRegLabels[reg] = a
		lz.atomicRegLabelsMux.Unlock()
	})

	return ""
}

// RegBitwidth returns the bitwidth of the queried register
func (lz *LazyCPU) RegBitwidth(reg registers.Generic) int {
	if lz.val.Dbg == nil {
		return 0
	}

	// label of register will neved change to if value is in atomicRegBitwidth
	// map we don't need to read it again

	lz.atomicRegBitwidthMux.RLock()
	defer lz.atomicRegBitwidthMux.RUnlock()
	if v, ok := lz.atomicRegBitwidth[reg]; ok {
		return v.Load().(int)
	}

	lz.val.Dbg.PushRawEvent(func() {
		var a atomic.Value
		a.Store(reg.BitWidth())
		lz.atomicRegBitwidthMux.Lock()
		lz.atomicRegBitwidth[reg] = a
		lz.atomicRegBitwidthMux.Unlock()
	})

	return 0
}

// RegValue returns the value for the queried register in hexadecimal
// string format. Note that a numeric representation of the PC register can be
// accessed through PCaddr
func (lz *LazyCPU) RegValue(reg registers.Generic) string {
	if lz.val.Dbg == nil {
		return ""
	}

	// value of register can change so we push a read event every instance.
	// this is unlike the RegLabel() and RegBitwidth() functions

	lz.val.Dbg.PushRawEvent(func() {
		var a atomic.Value
		a.Store(reg.String())
		lz.atomicRegValuesMux.Lock()
		lz.atomicRegValues[reg] = a
		lz.atomicRegValuesMux.Unlock()
	})

	lz.atomicRegValuesMux.RLock()
	defer lz.atomicRegValuesMux.RUnlock()
	if v, ok := lz.atomicRegValues[reg]; ok {
		return v.Load().(string)
	}

	return ""
}
