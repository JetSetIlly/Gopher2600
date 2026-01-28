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
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
)

// PushFunction onto the event queue. Used to ensure that the events are
// inserted into the emulation loop correctly.
func (dbg *Debugger) PushFunction(f func()) {
	select {
	case dbg.events.PushedFunction <- f:
	default:
		logger.Log(logger.Allow, "debugger", "dropped raw event push")
	}
}

// PushFunctionImmediate is the same as PushFunction but the event handler will
// return to the input loop for immediate action.
func (dbg *Debugger) PushFunctionImmediate(f func()) {
	select {
	case dbg.events.PushedFunctionImmediate <- f:
	default:
		logger.Log(logger.Allow, "debugger", "dropped raw event push (to return channel)")
	}
}

// PushSetMode sets the mode of the emulation.
func (dbg *Debugger) PushSetMode(mode govern.Mode) {
	dbg.PushFunctionImmediate(func() {
		dbg.setMode(mode)
	})
}

// PushSetPause sets the pause state of the emulation.
func (dbg *Debugger) PushSetPause(paused bool) {
	switch dbg.Mode() {
	case govern.ModePlay:
		dbg.PushFunction(func() {
			if paused {
				dbg.setState(govern.Paused, govern.Normal)
			} else {
				dbg.setState(govern.Running, govern.Normal)
			}
		})
	case govern.ModeDebugger:
		logger.Log(logger.Allow, "debugger", "not reacting to SetPause() in debugger mode (use terminal input instead)")
	}
}

// PushMemoryProfile forces a garbage collection event and takes a runtime
// memory profile and saves it to the working directory
func (dbg *Debugger) PushMemoryProfile() {
	dbg.PushFunctionImmediate(func() {
		fn, err := dbg.memoryProfile()
		if err != nil {
			logger.Log(logger.Allow, "memory profiling", err)
			return
		}
		logger.Logf(logger.Allow, "memory profiling", "saved to %s", fn)
	})
}

// PushReload re-inserts the current cartridge and restarts the emulation
func (dbg *Debugger) PushReload(end chan<- bool) {
	if dbg.Mode() == govern.ModeDebugger {
		dbg.PushFunctionImmediate(func() {
			// resetting in the middle of a CPU instruction requires the input loop to be unwound before continuing
			dbg.unwindLoop(func() error {
				err := dbg.reloadCartridge()
				if err != nil {
					return err
				}
				if end != nil {
					end <- true
				}
				return nil
			})
		})
	} else {
		dbg.PushFunction(func() {
			dbg.reloadCartridge()
		})
	}
}

// PushNotify implements the notifications.Notify interface
func (dbg *Debugger) PushNotify(notice notifications.Notice, data ...string) error {
	dbg.PushFunction(func() {
		dbg.Notify(notice, data...)
	})
	return nil
}
