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
			entries: make(map[string]IllegalAccessEntry),
			Log:     make([]IllegalAccessEntry, 0),
		},
	}

	dev.cart.SetDeveloper(dev)

	dev.source, err = NewSource(pathToROM)
	if err != nil {
		logger.Logf("developer", err.Error())
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
func (dev *Developer) IllegalAccess(event string, pc uint32, addr uint32) string {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	accessKey := fmt.Sprintf("%08x%08x", addr, pc)
	if e, ok := dev.illegalAccess.entries[accessKey]; ok {
		e.SrcLine.IllegalCount++
		return ""
	}

	e := IllegalAccessEntry{
		Event:      event,
		PC:         pc,
		AccessAddr: addr,
	}

	if dev.source != nil {
		var err error

		e.SrcLine, err = dev.source.findSourceLine(pc)
		if err != nil {
			logger.Logf("developer", "%v", err)
			return UnknownSourceLine
		}
	}

	dev.illegalAccess.entries[accessKey] = e
	dev.illegalAccess.Log = append(dev.illegalAccess.Log, e)

	if e.SrcLine == nil {
		return UnknownSourceLine
	}

	e.SrcLine.IllegalCount++

	return fmt.Sprintf("%s %s\n%s", e.SrcLine.String(), e.SrcLine.Function.Name, e.SrcLine.Content)
}

// ExecutionProfile implements the CartCoProcDeveloper interface.
func (dev *Developer) ExecutionProfile(addr map[uint32]float32) {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	if dev.source != nil {
		for pc, ct := range addr {
			dev.source.execute(pc, ct)
		}

		dev.source.SortedLines.Sort()
		dev.source.SortedFunctions.Sort()
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

	// traverse the SortedLines list and update the FrameCyles values
	//
	// we prefer this over traversing the Lines list because we may hit a
	// SourceLine more than once. SortedLines contains unique entries.
	for _, l := range dev.source.SortedLines.Lines {
		l.Stats.Update()
	}

	for _, f := range dev.source.Functions {
		f.Stats.Update()
	}

	dev.source.Stats.Update()

	return nil
}
