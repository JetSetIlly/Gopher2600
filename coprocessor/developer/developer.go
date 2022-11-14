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
	"sync"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/logger"
)

// Developer implements the CartCoProcDeveloper interface.
type Developer struct {
	cart CartCoProcDeveloper
	tv   TV

	// only respond on the CartCoProcDeveloper interface when enabled
	disabledExpensive bool

	// information about the source code to the program. can be nil.
	// note that source is checked for nil outside the sourceLock. this is
	// performance reasons (not need to acquire the lock if source is nil).
	// however, this does mean we should be careful if reassigning the source
	// field (but that doesn't happen)
	source     *Source
	sourceLock sync.Mutex

	// illegal accesses already encountered. duplicate accesses will not be logged.
	illegalAccess     IllegalAccess
	illegalAccessLock sync.Mutex

	// the current yield information
	yieldState     YieldState
	yieldStateLock sync.Mutex

	framesSinceLastUpdate int

	// profiler instance. measures cycles counts for executed address
	profiler mapper.CartCoProcProfiler

	// frame info from the last NewFrame()
	frameInfo television.FrameInfo
}

// TV is the interface from the developer type to the television implementation.
type TV interface {
	GetFrameInfo() television.FrameInfo
	GetCoords() coords.TelevisionCoords
}

// CartCoProcDeveloper defines the interface to the cartridge required by the
// developer pacakge
type CartCoProcDeveloper interface {
	GetCoProc() mapper.CartCoProc
	PushFunction(func())
}

// NewDeveloper is the preferred method of initialisation for the Developer type.
func NewDeveloper(romFile string, cart CartCoProcDeveloper, tv TV, elfFile string) *Developer {
	if cart == nil || cart.GetCoProc() == nil {
		return nil
	}

	var err error

	dev := &Developer{
		cart: cart,
		tv:   tv,
		illegalAccess: IllegalAccess{
			entries: make(map[string]*IllegalAccessEntry),
			Log:     make([]*IllegalAccessEntry, 0),
		},
		profiler: mapper.CartCoProcProfiler{
			Entries: make([]mapper.CartCoProcProfileEntry, 0, 1000),
		},
	}

	t := time.Now()
	dev.source, err = NewSource(romFile, cart, elfFile)
	if err != nil {
		logger.Logf("developer", err.Error())
	} else {
		logger.Logf("developer", "DWARF loaded in %s", time.Since(t))
	}

	if dev.source != nil {
		dev.source.SortedLines.SortByLineAndFunction(false)
		dev.source.SortedFunctions.SortByFunction(false)
	}

	// we always set the devloper for the cartridge even if we have no source.
	// some developer functions don't require source code to be useful
	dev.cart.GetCoProc().SetDeveloper(dev)

	return dev
}

// Strings used to indicate unknown values.
const (
	UnknownFunction   = "<unknown function>"
	UnknownSourceLine = "<unknown source line>"
)

// DisableExpensive prevents the computationaly expensive developer functions
// from running.
func (dev *Developer) DisableExpensive(disable bool) {
	dev.disabledExpensive = disable
}

// VariableMemtop implements the CartCoProcDeveloper interface.
func (dev *Developer) VariableMemtop() uint32 {
	if dev.source == nil {
		return 0
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	return uint32(dev.source.VariableMemtop)
}

// CheckBreakpoint implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) CheckBreakpoint(addr uint32) bool {
	if dev.disabledExpensive {
		return false
	}

	if dev.source == nil {
		return false
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	return dev.source.CheckBreakpoint(addr)
}

// HasSource returns true if source information has been found.
func (dev *Developer) HasSource() bool {
	return dev.source != nil
}

const maxWaitUpdateTime = 60 // in frames

// NewFrame implements the television.FrameTrigger interface.
func (dev *Developer) NewFrame(frameInfo television.FrameInfo) error {
	// only update FrameCycles if new frame was caused by a VSYNC or we've
	// waited long enough since the last update
	dev.framesSinceLastUpdate++
	if !frameInfo.VSync || dev.framesSinceLastUpdate > maxWaitUpdateTime {
		return nil
	}
	dev.framesSinceLastUpdate = 0

	// do nothing else if no source is available
	if dev.source == nil {
		return nil
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	dev.source.newFrame()
	dev.frameInfo = frameInfo

	return nil
}

// ResetStatistics resets all performance statistics. This differs from the
// function in the Source type in that it acquires and releases the source
// critical section.
func (dev *Developer) ResetStatistics() {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	dev.source.ResetStatistics()
}
