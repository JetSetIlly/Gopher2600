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

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/breakpoints"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/callstack"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/faults"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/yield"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/logger"
)

// TV is the interface from the developer type to the television implementation.
type TV interface {
	GetFrameInfo() television.FrameInfo
	GetCoords() coords.TelevisionCoords
	GetLastSignal() signal.SignalAttributes
}

// Cartridge defines the interface to the cartridge required by the developer package
type Cartridge interface {
	GetCoProcBus() coprocessor.CartCoProcBus
	PushFunction(func())
}

// Emulation defines an interface to the emulation for retreiving the emulation state
type Emulation interface {
	State() govern.State
}

// Developer implements the CartCoProcDeveloper interface.
type Developer struct {
	emulation Emulation
	tv        TV

	cart Cartridge

	// information about the source code to the program. can be nil.
	// note that source is checked for nil outside the sourceLock. this is
	// performance reasons (not need to acquire the lock if source is nil).
	// however, this does mean we should be careful if reassigning the source
	// field (but that doesn't happen)
	source     *dwarf.Source
	sourceLock sync.Mutex

	faults     faults.Faults
	faultsLock sync.Mutex

	yieldState     yield.State
	yieldStateLock sync.Mutex

	callstack     callstack.CallStack
	callstackLock sync.Mutex

	breakpoints          breakpoints.Breakpoints
	breakpointsLock      sync.Mutex
	breakNextInstruction bool
	breakAddress         uint32

	// profiler instance. measures cycles counts for executed address
	profiler coprocessor.CartCoProcProfiler

	// slow down rate of NewFrame()
	framesSinceLastUpdate int

	// keeps track of the previous breakpoint check. see checkBreakPointByAddr()
	prevBreakpointCheck *dwarf.SourceLine

	// keeps track of the previous line in profiling scan. see processProfiling()
	prevProfileLine *dwarf.SourceLine
}

// NewDeveloper is the preferred method of initialisation for the Developer type.
func NewDeveloper(state Emulation, tv TV) Developer {
	return Developer{
		emulation: state,
		tv:        tv,
		yieldState: yield.State{
			StrobeTicker: time.NewTicker(100 * time.Microsecond),
		},
	}
}

func (dev *Developer) AttachCartridge(cart Cartridge, romFile string, elfFile string) error {
	dev.cart = nil

	dev.sourceLock.Lock()
	dev.source = nil
	dev.sourceLock.Unlock()

	dev.faultsLock.Lock()
	dev.faults = faults.NewFaults()
	dev.faultsLock.Unlock()

	dev.callstackLock.Lock()
	dev.callstack = callstack.NewCallStack()
	dev.callstackLock.Unlock()

	dev.breakpointsLock.Lock()
	dev.breakpoints = breakpoints.NewBreakpoints()
	dev.breakpointsLock.Unlock()

	dev.framesSinceLastUpdate = 0

	dev.profiler = coprocessor.CartCoProcProfiler{
		Entries: make([]coprocessor.CartCoProcProfileEntry, 0, 1000),
	}

	if cart == nil || cart.GetCoProcBus() == nil {
		return nil
	}
	dev.cart = cart

	// we always set the developer for the cartridge even if we have no source.
	// some developer functions don't require source code to be useful
	dev.cart.GetCoProcBus().GetCoProc().SetDeveloper(dev)

	switch dev.emulation.State() {
	case govern.EmulatorStart:
	case govern.Initialising:
	default:
		panic("unexpected emulation on cartridge insertion")
	}

	var err error

	t := time.Now()

	dev.sourceLock.Lock()
	dev.source, err = dwarf.NewSource(romFile, cart, elfFile)
	dev.sourceLock.Unlock()

	if err != nil {
		return fmt.Errorf("developer: %w", err)
	} else {
		logger.Logf(logger.Allow, "developer", "DWARF loaded in %s", time.Since(t))
	}

	return nil
}

// HighAddress implements the coprocessor.CartCoProcDeveloper interface.
func (dev *Developer) HighAddress() uint32 {
	if dev.source == nil {
		return 0
	}

	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	return uint32(dev.source.HighAddress)
}

// CheckBreakpoint implements the coprocessor.CartCoProcDeveloper interface.
func (dev *Developer) CheckBreakpoint(addr uint32) bool {
	if dev.source == nil {
		return false
	}

	if dev.breakNextInstruction && dev.breakAddress != addr {
		dev.breakNextInstruction = false
		dev.breakAddress = addr
		return true
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

	if dev.breakpoints.Check(addr) {
		dev.breakAddress = addr
		return true
	}
	return false
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
	dev.source.NewFrame(dev.emulation.State() == govern.Rewinding)

	return nil
}

// EnableStrobe sets the yield strobe to the specified enable state and the
// specified address
func (dev *Developer) EnableStrobe(enable bool, addr uint32) {
	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()
	dev.yieldState.Strobe = enable
	dev.yieldState.StrobeAddr = addr
}

// OnYield implements the coprocessor.CartCoProcDeveloper interface.
func (dev *Developer) OnYield(addr uint32, yield coprocessor.CoProcYield) {
	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()

	dev.yieldState.Addr = addr
	dev.yieldState.Reason = yield.Type
	dev.yieldState.LocalVariables = dev.yieldState.LocalVariables[:0]

	// buildLocalsList() depends on the type of yield and whether a strobe is active
	buildLocalsList := func(locals []*dwarf.SourceVariableLocal) {
		dev.yieldState.LocalVariables = append(dev.yieldState.LocalVariables, locals...)
	}

	if yield.Type == coprocessor.YieldSyncWithVCS {
		if !dev.yieldState.Strobe || addr != dev.yieldState.StrobeAddr {
			return
		}

		select {
		case <-dev.yieldState.StrobeTicker.C:
			buildLocalsList = func(locals []*dwarf.SourceVariableLocal) {
				dev.yieldState.StrobedLocalVariables = dev.yieldState.StrobedLocalVariables[:0]
				dev.yieldState.StrobedLocalVariables = append(dev.yieldState.StrobedLocalVariables, locals...)
			}
		default:
			return
		}
	}

	dev.BorrowSource(func(src *dwarf.Source) {
		if src == nil {
			return
		}

		ln := src.FindSourceLine(dev.yieldState.Addr)
		if ln == nil {
			return
		}

		locals := src.GetLocalVariables(ln, addr)
		buildLocalsList(locals)

		if yield.Type.Bug() {
			ln.Bug = true
		}

		// update local variables
		for _, local := range locals {
			local.Update()
		}

		src.UpdateGlobalVariables()
	})
}

// MemoryFault implements the coprocessor.CartCoProcDeveloper interface.
func (dev *Developer) MemoryFault(event string, fault faults.Category, instructionAddr uint32, accessAddr uint32) {
	dev.faultsLock.Lock()
	defer dev.faultsLock.Unlock()

	dev.faults.NewEntry(fault, event, instructionAddr, accessAddr)
}

// SetEmulationState is called by the emulation whenever state changes
func (dev *Developer) SetEmulationState(state govern.State) {
	if dev.cart != nil {
		switch state {
		case govern.Rewinding:
			dev.cart.GetCoProcBus().GetCoProc().BreakpointsEnable(false)
		default:
			dev.cart.GetCoProcBus().GetCoProc().BreakpointsEnable(true)
		}
	}

	dev.BorrowSource(func(src *dwarf.Source) {
		dev.yieldStateLock.Lock()
		defer dev.yieldStateLock.Unlock()

		switch state {
		case govern.Paused:
			if src == nil {
				return
			}

			ln := src.FindSourceLine(dev.yieldState.Addr)
			if ln == nil {
				return
			}

			locals := src.GetLocalVariables(ln, dev.yieldState.Addr)

			dev.yieldState.LocalVariables = dev.yieldState.LocalVariables[:0]
			dev.yieldState.LocalVariables = append(dev.yieldState.LocalVariables, locals...)

			src.UpdateGlobalVariables()
		default:
			dev.yieldState.LocalVariables = dev.yieldState.LocalVariables[:0]
		}
	})
}

// BreakOnNextStep forces the coprocess to break after next instruction execution
func (dev *Developer) BreakNextInstruction() {
	dev.breakNextInstruction = true
}
