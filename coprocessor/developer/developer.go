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

	"github.com/jetsetilly/gopher2600/coprocessor/developer/breakpoints"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/callstack"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/yield"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/logger"
)

// TV is the interface from the developer type to the television implementation.
type TV interface {
	GetFrameInfo() television.FrameInfo
	GetCoords() coords.TelevisionCoords
}

// Cartridge defines the interface to the cartridge required by the developer package
type Cartridge interface {
	GetCoProc() mapper.CartCoProc
	PushFunction(func())
}

// Developer implements the CartCoProcDeveloper interface.
type Developer struct {
	cart Cartridge
	tv   TV

	// only respond on the CartCoProcDeveloper interface when enabled
	disabledExpensive bool

	// information about the source code to the program. can be nil.
	// note that source is checked for nil outside the sourceLock. this is
	// performance reasons (not need to acquire the lock if source is nil).
	// however, this does mean we should be careful if reassigning the source
	// field (but that doesn't happen)
	source     *dwarf.Source
	sourceLock sync.Mutex

	// illegal accesses already encountered. duplicate accesses will not be logged.
	illegalAccess     IllegalAccess
	illegalAccessLock sync.Mutex

	// the current yield information
	yieldState     yield.State
	yieldStateLock sync.Mutex

	callstack     callstack.CallStack
	callstackLock sync.Mutex

	breakpoints     breakpoints.Breakpoints
	breakpointsLock sync.Mutex

	// slow down rate of NewFrame()
	framesSinceLastUpdate int

	// profiler instance. measures cycles counts for executed address
	profiler mapper.CartCoProcProfiler

	// frame info from the last NewFrame()
	frameInfo television.FrameInfo

	// keeps track of the previous breakpoint check. see checkBreakPointByAddr()
	prevBreakpointCheck *dwarf.SourceLine
}

// NewDeveloper is the preferred method of initialisation for the Developer type.
func NewDeveloper(tv TV) Developer {
	return Developer{
		tv: tv,
		callstack: callstack.CallStack{
			Callers: make(map[string][]*dwarf.SourceLine),
		},
		breakpoints: breakpoints.NewBreakpoints(),
	}
}

func (dev *Developer) AttachCartridge(cart Cartridge, romFile string, elfFile string) {
	dev.cart = nil

	dev.sourceLock.Lock()
	dev.source = nil
	dev.sourceLock.Unlock()

	dev.disabledExpensive = false

	dev.illegalAccessLock.Lock()
	dev.illegalAccess = IllegalAccess{
		entries: make(map[string]*IllegalAccessEntry),
	}
	dev.illegalAccessLock.Unlock()

	dev.yieldStateLock.Lock()
	dev.yieldState = yield.State{}
	dev.yieldStateLock.Unlock()

	dev.framesSinceLastUpdate = 0

	dev.profiler = mapper.CartCoProcProfiler{
		Entries: make([]mapper.CartCoProcProfileEntry, 0, 1000),
	}

	if cart == nil || cart.GetCoProc() == nil {
		return
	}
	dev.cart = cart

	var err error

	t := time.Now()

	dev.sourceLock.Lock()
	dev.source, err = dwarf.NewSource(romFile, cart, elfFile)
	dev.sourceLock.Unlock()

	if err != nil {
		logger.Logf("developer", err.Error())
	} else {
		logger.Logf("developer", "DWARF loaded in %s", time.Since(t))
	}

	// we always set the developer for the cartridge even if we have no source.
	// some developer functions don't require source code to be useful
	dev.cart.GetCoProc().SetDeveloper(dev)
}

// DisableExpensive prevents the computationaly expensive developer functions
// from running.
func (dev *Developer) DisableExpensive(disable bool) {
	dev.disabledExpensive = disable
}

// HighAddress implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) HighAddress() uint32 {
	if dev.source == nil {
		return 0
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	return uint32(dev.source.HighAddress)
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

	ln := dev.source.LinesByAddress[uint64(addr)]
	if ln == dev.prevBreakpointCheck {
		return false
	}

	dev.prevBreakpointCheck = ln

	dev.breakpointsLock.Lock()
	defer dev.breakpointsLock.Unlock()

	return dev.breakpoints.Check(addr)
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

	dev.source.NewFrame()
	dev.frameInfo = frameInfo

	return nil
}

// OnYield implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) OnYield(addr uint32, yield mapper.CoProcYield) {
	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()

	dev.yieldState.Addr = addr
	dev.yieldState.Reason = yield.Type
	dev.yieldState.LocalVariables = dev.yieldState.LocalVariables[:0]

	if yield.Type == mapper.YieldSyncWithVCS {
		return
	}

	dev.BorrowSource(func(src *dwarf.Source) {
		if src == nil {
			return
		}

		locals := src.OnYield(addr, yield)
		dev.yieldState.LocalVariables = append(dev.yieldState.LocalVariables, locals...)
	})
}
