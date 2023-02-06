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

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

// LazyValues contains all values required by a debugger running in a different
// thread to the emulation. Use these values rather than directly accessing
// those exposed by the emulation.
type LazyValues struct {
	dbg *debugger.Debugger
	tv  *television.Television
	vcs *hardware.VCS

	// pointers to these instances. non-pointer instances trigger the race
	// detector for some reason.
	Debugger    *LazyDebugger
	CPU         *LazyCPU
	Bus         *LazyBus
	Phaseclock  *LazyPhaseClock
	RAM         *LazyRAM
	Timer       *LazyTimer
	Playfield   *LazyPlayfield
	Player0     *LazyPlayer
	Player1     *LazyPlayer
	Missile0    *LazyMissile
	Missile1    *LazyMissile
	Ball        *LazyBall
	TV          *LazyTV
	Cart        *LazyCart
	Peripherals *LazyPeripherals
	Collisions  *LazyCollisions
	Ports       *LazyPorts
	Tracker     *LazyTracker
	SaveKey     *LazySaveKey
	Rewind      *LazyRewind

	// we need a way of making sure we don't update the lazy values too often.
	// if we're not careful the GUI thread can push refresh requests more
	// quickly than the debugger input loop can handel them. this is
	// particularly noticeable during a REWIND or GOTO event
	refreshScheduled atomic.Value
	refreshDone      atomic.Value
}

// NewLazyValues is the preferred method of initialisation for the Values type.
func NewLazyValues(dbg *debugger.Debugger) *LazyValues {
	val := &LazyValues{}

	val.dbg = dbg
	val.tv = val.dbg.TV()
	val.vcs = val.dbg.VCS()

	val.Debugger = newLazyDebugger(val)
	val.CPU = newLazyCPU(val)
	val.Bus = newLazyBus(val)
	val.Phaseclock = newLazyPhaseClock(val)
	val.RAM = newLazyRAM(val)
	val.Timer = newLazyTimer(val)
	val.Playfield = newLazyPlayfield(val)
	val.Player0 = newLazyPlayer(val, 0)
	val.Player1 = newLazyPlayer(val, 1)
	val.Missile0 = newLazyMissile(val, 0)
	val.Missile1 = newLazyMissile(val, 1)
	val.Ball = newLazyBall(val)
	val.TV = newLazyTV(val)
	val.Cart = newLazyCart(val)
	val.Peripherals = newLazyPeripherals(val)
	val.Collisions = newLazyCollisions(val)
	val.Ports = newLazyPorts(val)
	val.Tracker = newLazyTracker(val)
	val.SaveKey = newLazySaveKey(val)
	val.Rewind = newLazyRewind(val)

	val.refreshScheduled.Store(false)
	val.refreshDone.Store(false)

	return val
}

// Refresh lazy values.
func (val *LazyValues) Refresh() {
	if val.dbg == nil {
		return
	}

	if val.refreshDone.Load().(bool) {
		val.refreshDone.Store(false)

		val.Debugger.update()
		val.CPU.update()
		val.Bus.update()
		val.Phaseclock.update()
		val.RAM.update()
		val.Timer.update()
		val.Playfield.update()
		val.Player0.update()
		val.Player1.update()
		val.Missile0.update()
		val.Missile1.update()
		val.Ball.update()
		val.TV.update()
		val.Cart.update()
		val.Peripherals.update()
		val.Collisions.update()
		val.Ports.update()
		val.Tracker.update()
		val.SaveKey.update()
		val.Rewind.update()
	}

	if val.refreshScheduled.Load().(bool) {
		return
	}
	val.refreshScheduled.Store(true)

	val.dbg.PushFunction(func() {
		val.Debugger.push()
		val.CPU.push()
		val.Bus.push()
		val.Phaseclock.push()
		val.RAM.push()
		val.Timer.push()
		val.Playfield.push()
		val.Player0.push()
		val.Player1.push()
		val.Missile0.push()
		val.Missile1.push()
		val.Ball.push()
		val.TV.push()
		val.Cart.push()
		val.Peripherals.push()
		val.Collisions.push()
		val.Ports.push()
		val.Tracker.push()
		val.SaveKey.push()
		val.Rewind.push()
		val.refreshScheduled.Store(false)
		val.refreshDone.Store(true)
	})
}

// FastRefresh lazy values. Updates only the values that are needed in playmode.
func (val *LazyValues) FastRefresh() {
	if val.dbg == nil {
		return
	}

	if val.refreshDone.Load().(bool) {
		val.refreshDone.Store(false)
		val.TV.update()
		val.Tracker.update()
		val.Cart.fastUpdate()
	}

	if val.refreshScheduled.Load().(bool) {
		return
	}
	val.refreshScheduled.Store(true)

	val.dbg.PushFunction(func() {
		val.TV.push()
		val.Tracker.push()
		val.Cart.fastPush()
		val.refreshScheduled.Store(false)
		val.refreshDone.Store(true)
	})
}
