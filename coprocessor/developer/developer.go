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
	"fmt"
	"sync"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
)

// Developer implements the CartCoProcDeveloper interface.
type Developer struct {
	cart mapper.CartCoProcBus

	// information about the source code to the program. can be nil
	source *Source

	// lock source
	sourceLock sync.Mutex

	// illegal accesses already encountered. duplicate accesses will not be logged.
	illegalAccess     IllegalAccess
	illegalAccessLock sync.Mutex

	framesSinceLastUpdate int
}

// NewDeveloper is the preferred method of initialisation for the Developer type.
func NewDeveloper(pathToROM string, cart mapper.CartCoProcBus) *Developer {
	if cart == nil {
		return nil
	}

	var err error

	dev := &Developer{
		cart: cart,
		illegalAccess: IllegalAccess{
			entries: make(map[string]*IllegalAccessEntry),
			Log:     make([]*IllegalAccessEntry, 0),
		},
	}

	dev.cart.SetDeveloper(dev)

	t := time.Now()
	dev.source, err = NewSource(pathToROM)
	if err != nil {
		logger.Logf("developer", err.Error())
	} else {
		logger.Logf("developer", "DWARF loaded in %s", time.Since(t))
	}

	if dev.source != nil {
		dev.source.SortedLines.SortByLineAndFunction(false)
		dev.source.SortedFunctions.SortByFunction(false)
	}

	return dev
}

// Strings used to indicate unknown values.
const (
	UnknownFunction   = "<unknown function>"
	UnknownSourceLine = "<unknown source line>"
)

// IllegalAccess implements the CartCoProcDeveloper interface.
func (dev *Developer) NullAccess(event string, pc uint32, addr uint32) string {
	return dev.logAccess(event, pc, addr, true)
}

// IllegalAccess implements the CartCoProcDeveloper interface.
func (dev *Developer) IllegalAccess(event string, pc uint32, addr uint32) string {
	return dev.logAccess(event, pc, addr, false)
}

// IllegalAccess implements the CartCoProcDeveloper interface.
func (dev *Developer) StackCollision(pc uint32, addr uint32) string {
	dev.illegalAccess.HasStackCollision = true
	return dev.logAccess("Stack Collision", pc, addr, false)
}

// VariableMemtop implements the CartCoProcDeveloper interface.
func (dev *Developer) VariableMemtop() uint32 {
	if dev.source == nil {
		return 0
	}
	return uint32(dev.source.VariableMemtop)
}

// logAccess adds an illegal or null access event to the log. includes source code lookup
func (dev *Developer) logAccess(event string, pc uint32, addr uint32, isNullAccess bool) string {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	// get/create illegal access entry
	accessKey := fmt.Sprintf("%08x%08x", addr, pc)
	e, ok := dev.illegalAccess.entries[accessKey]
	if ok {
		// we seen the illegal access before - increase count
		e.Count++
	} else {
		e = &IllegalAccessEntry{
			Event:        event,
			PC:           pc,
			AccessAddr:   addr,
			Count:        1,
			IsNullAccess: isNullAccess,
		}

		dev.illegalAccess.entries[accessKey] = e

		// we always log illegal accesses even if we don't have any source
		// information
		if dev.source != nil {
			var err error

			e.SrcLine, err = dev.source.findSourceLine(pc)
			if err != nil {
				logger.Logf("developer", "%v", err)
				return ""
			}

			// inidcate that the source line has been responsble for an illegal access
			if e.SrcLine != nil {
				e.SrcLine.IllegalAccess = true
			}
		}

		// record access
		dev.illegalAccess.entries[accessKey] = e

		// update log
		dev.illegalAccess.Log = append(dev.illegalAccess.Log, e)
	}

	// no source line information so return empty line
	if e.SrcLine == nil {
		return ""
	}

	return fmt.Sprintf("%s %s\n%s", e.SrcLine.String(), e.SrcLine.Function.Name, e.SrcLine.PlainContent)
}

// ExecutionProfile implements the CartCoProcDeveloper interface.
func (dev *Developer) ExecutionProfile(addr map[uint32]float32) {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	if dev.source != nil {
		for pc, ct := range addr {
			dev.source.executeProfile(pc, ct)
		}
	}
}

// HasSource returns true if source information has been found.
func (dev *Developer) HasSource() bool {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()
	return dev.source != nil
}

// BorrowSource will lock the source code structure for the durction of the
// supplied function, which will be executed with the source code structure as
// an argument.
//
// May return nil.
func (dev *Developer) BorrowSource(f func(*Source)) {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()
	f(dev.source)
}

// BorrowIllegalAccess will lock the illegal access log for the duration of the
// supplied fucntion, which will be executed with the illegal access log as an
// argument.
func (dev *Developer) BorrowIllegalAccess(f func(*IllegalAccess)) {
	dev.illegalAccessLock.Lock()
	defer dev.illegalAccessLock.Unlock()
	f(&dev.illegalAccess)
}

const maxWaitUpdateTime = 60 // in frames

// NewFrame implements the television.FrameTrigger interface.
func (dev *Developer) NewFrame(frameInfo television.FrameInfo) error {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	// only update FrameCycles if new frame was caused by a VSYNC or we've
	// waited long enough since the last update
	dev.framesSinceLastUpdate++
	if !frameInfo.VSynced || dev.framesSinceLastUpdate > maxWaitUpdateTime {
		return nil
	}
	dev.framesSinceLastUpdate = 0

	// do nothing else if no source is available
	if dev.source == nil {
		return nil
	}

	dev.source.newFrame()

	return nil
}
