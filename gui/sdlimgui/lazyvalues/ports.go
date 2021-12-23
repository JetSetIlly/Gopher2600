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

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
)

// LazyPorts lazily accesses RIOT Ports information from the emulator.
type LazyPorts struct {
	val *LazyValues

	swcha         atomic.Value // uint8
	swchb         atomic.Value // uint8
	swcha_w       atomic.Value // uint8
	swcha_derived atomic.Value // uint8
	swacnt        atomic.Value // uint8
	swbcnt        atomic.Value // uint8
	swchb_w       atomic.Value // uint8
	swchb_derived atomic.Value // uint8
	inpt0         atomic.Value // uint8
	inpt1         atomic.Value // uint8
	inpt2         atomic.Value // uint8
	inpt3         atomic.Value // uint8
	inpt4         atomic.Value // uint8
	inpt5         atomic.Value // uint8

	SWCHA         uint8
	SWACNT        uint8
	SWCHA_W       uint8
	SWCHA_Derived uint8
	SWCHB         uint8
	SWBCNT        uint8
	SWCHB_W       uint8
	SWCHB_Derived uint8
	INPT0         uint8
	INPT1         uint8
	INPT2         uint8
	INPT3         uint8
	INPT4         uint8
	INPT5         uint8
}

func newLazyPorts(val *LazyValues) *LazyPorts {
	return &LazyPorts{val: val}
}

func (lz *LazyPorts) push() {
	v := lz.val.vcs.RIOT.Ports.GetField("swcha")
	lz.swcha.Store(v)
	v = lz.val.vcs.RIOT.Ports.GetField("swacnt")
	lz.swacnt.Store(v)
	v = lz.val.vcs.RIOT.Ports.GetField("swcha_w")
	lz.swcha_w.Store(v)
	v = lz.val.vcs.RIOT.Ports.GetField("swcha_derived")
	lz.swcha_derived.Store(v)

	v = lz.val.vcs.RIOT.Ports.GetField("swchb")
	lz.swchb.Store(v)
	v = lz.val.vcs.RIOT.Ports.GetField("swbcnt")
	lz.swbcnt.Store(v)
	v = lz.val.vcs.RIOT.Ports.GetField("swchb_w")
	lz.swchb_w.Store(v)
	v = lz.val.vcs.RIOT.Ports.GetField("swchb_derived")
	lz.swchb_derived.Store(v)

	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["INPT0"])
	lz.inpt0.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["INPT1"])
	lz.inpt1.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["INPT2"])
	lz.inpt2.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["INPT3"])
	lz.inpt3.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["INPT4"])
	lz.inpt4.Store(v)
	v, _ = lz.val.vcs.Mem.Peek(addresses.ReadAddress["INPT5"])
	lz.inpt5.Store(v)
}

func (lz *LazyPorts) update() {
	lz.SWCHA, _ = lz.swcha.Load().(uint8)
	lz.SWACNT, _ = lz.swacnt.Load().(uint8)
	lz.SWCHA_W, _ = lz.swcha_w.Load().(uint8)
	lz.SWCHA_Derived, _ = lz.swcha_derived.Load().(uint8)

	lz.SWCHB, _ = lz.swchb.Load().(uint8)
	lz.SWBCNT, _ = lz.swbcnt.Load().(uint8)
	lz.SWCHB_W, _ = lz.swchb_w.Load().(uint8)
	lz.SWCHB_Derived, _ = lz.swchb_derived.Load().(uint8)

	lz.INPT0, _ = lz.inpt0.Load().(uint8)
	lz.INPT1, _ = lz.inpt1.Load().(uint8)
	lz.INPT2, _ = lz.inpt2.Load().(uint8)
	lz.INPT3, _ = lz.inpt3.Load().(uint8)
	lz.INPT4, _ = lz.inpt4.Load().(uint8)
	lz.INPT5, _ = lz.inpt5.Load().(uint8)
}
