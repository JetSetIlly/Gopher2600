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

package developer

import (
	"github.com/jetsetilly/gopher2600/coprocessor/developer/breakpoints"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/callstack"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/faults"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/yield"
)

// BorrowSource will lock the source code structure for the durction of the
// supplied function, which will be executed with the source code structure as
// an argument.
//
// May return nil.
func (dev *Developer) BorrowSource(f func(*dwarf.Source)) {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()
	f(dev.source)
}

// BorrowCallStack will lock the callstack structure for the durction of the
// supplied function, which will be executed with the callstack structure as
// an argument.
//
// In some situations BorrowCallStack may need to be wrapped in a BorrowSource
// call in order to prevent a race condition
func (dev *Developer) BorrowCallStack(f func(callstack.CallStack)) {
	dev.callstackLock.Lock()
	defer dev.callstackLock.Unlock()
	f(dev.callstack)
}

// BorrowBreakpoints will lock the breakpoints structure for the durction of the
// supplied function, which will be executed with the breakpoints structure as
// an argument.
func (dev *Developer) BorrowBreakpoints(f func(breakpoints.Breakpoints)) {
	dev.breakpointsLock.Lock()
	defer dev.breakpointsLock.Unlock()
	f(dev.breakpoints)
}

// BorrowYieldState will lock the illegal access log for the duration of the
// supplied fucntion, which will be executed with the illegal access log as an
// argument.
//
// In some situations BorrowYieldState may need to be wrapped in a BorrowSource
// call in order to prevent a race condition
func (dev *Developer) BorrowYieldState(f func(yield.State)) {
	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()
	f(dev.yieldState)
}

// BorrowFaults will lock the illegal access log for the duration of the
// supplied fucntion, which will be executed with the illegal access log as an
// argument.
func (dev *Developer) BorrowFaults(f func(*faults.Faults)) {
	dev.faultsLock.Lock()
	defer dev.faultsLock.Unlock()
	f(&dev.faults)
}
