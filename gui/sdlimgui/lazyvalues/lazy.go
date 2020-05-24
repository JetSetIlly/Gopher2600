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
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// Lazy contains all values required by a debugger running in a different
// thread to the emulation. Use these values rather than directly accessing
// those exposed by the emulation.
type Lazy struct {
	// these fields are racy, they should not be accessed except through the
	// lazy evaluation system
	Dbg *debugger.Debugger

	// pointers to these instances. non-pointer instances trigger the race
	// detector for some reason.
	Debugger   *LazyDebugger
	CPU        *LazyCPU
	Timer      *LazyTimer
	Playfield  *LazyPlayfield
	Player0    *LazyPlayer
	Player1    *LazyPlayer
	Missile0   *LazyMissile
	Missile1   *LazyMissile
	Ball       *LazyBall
	TV         *LazyTV
	Cart       *LazyCart
	Controller *LazyControllers
	Prefs      *LazyPrefs

	// \/\/\/ the following are updated on demand rather than through the update
	// function, because they require more context
	//
	// there are no corresponding, non-atomic values for these slices. instead
	// use the corresponding functions function to update and retrieve on
	// demand \/\/\/

	// note that we use atomicRAM for both internal VCS ram and any additional
	// cartridge ram. as it is, internal RAM and each cartridge RAM bank are
	// never on screen at the same time so for display purposes we don't need
	// to distinguish between the different areas.
	atomicRAM []atomic.Value // []uint8

	// breakpoints
	atomicBrk []atomic.Value // debugger.BreakGroup
}

// NewValues is the preferred method of initialisation for the Values type
func NewValues() *Lazy {
	val := &Lazy{}

	val.Debugger = newLazyDebugger(val)
	val.CPU = newLazyCPU(val)
	val.Timer = newLazyTimer(val)
	val.Playfield = newLazyPlayfield(val)
	val.Player0 = newLazyPlayer(val, 0)
	val.Player1 = newLazyPlayer(val, 1)
	val.Missile0 = newLazyMissile(val, 0)
	val.Missile1 = newLazyMissile(val, 1)
	val.Ball = newLazyBall(val)
	val.TV = newLazyTV(val)
	val.Cart = newLazyCart(val)
	val.Controller = newLazyControllers(val)
	val.Prefs = newLazyPrefs(val)

	// allocating enough ram for an entire cart bank because, theoretically, a
	// cartridge format could have a RAM area as large as that
	val.atomicRAM = make([]atomic.Value, memorymap.MemtopCart-memorymap.OriginCart+1)

	// allocating enough space for every byte in cartridge space. not worrying
	// about bank sizes or anything like that.
	val.atomicBrk = make([]atomic.Value, memorymap.MemtopCart-memorymap.OriginCart+1)

	return val
}

// Update lazy values, with the exception of RAM and break information.
func (val *Lazy) Update() {
	if val.Dbg == nil {
		return
	}

	val.Debugger.update()
	val.CPU.update()
	val.Timer.update()
	val.Playfield.update()
	val.Player0.update()
	val.Player1.update()
	val.Missile0.update()
	val.Missile1.update()
	val.Ball.update()
	val.TV.update()
	val.Cart.update()
	val.Controller.update()
	val.Prefs.update()
}

// ReadRAM returns the data at read address
func (val *Lazy) ReadRAM(ramDetails memorymap.SubArea, readAddr uint16) uint8 {
	if val.Dbg == nil {
		return 0
	}

	val.Dbg.PushRawEvent(func() {
		d, _ := val.Dbg.VCS.Mem.Read(readAddr)
		val.atomicRAM[readAddr^ramDetails.ReadOrigin].Store(d)
	})
	d, _ := val.atomicRAM[readAddr^ramDetails.ReadOrigin].Load().(uint8)
	return d
}

// HasBreak checks to see if disassembly entry has a break point
func (val *Lazy) HasBreak(e *disassembly.Entry) debugger.BreakGroup {
	if val.Dbg == nil {
		return debugger.BrkNone
	}

	addr := e.Result.Address & memorymap.AddressMaskCart

	val.Dbg.PushRawEvent(func() {
		val.atomicBrk[addr].Store(val.Dbg.HasBreak(e))
	})

	if b, ok := val.atomicBrk[addr].Load().(debugger.BreakGroup); ok {
		return b
	}

	return debugger.BrkNone
}
