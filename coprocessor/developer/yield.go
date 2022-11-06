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
	"time"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// YieldState records the most recent yield.
type YieldState struct {
	InstructionPC uint32
	Reason        mapper.YieldReason
	TimeStamp     time.Time
}

// OnYield implements the mapper.CartCoProcDeveloper interface.
func (dev *Developer) OnYield(instructionPC uint32, reason mapper.YieldReason) {
	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()

	dev.yieldState.InstructionPC = instructionPC
	dev.yieldState.Reason = reason
	dev.yieldState.TimeStamp = time.Now()

	switch reason {
	case mapper.YieldMemoryAccessError:
		fallthrough
	case mapper.YieldExecutionError:
		fallthrough
	case mapper.YieldUnimplementedFeature:
		fallthrough
	case mapper.YieldUndefinedBehaviour:
		if dev.source != nil {
			dev.sourceLock.Lock()
			ln := dev.source.linesByAddress[uint64(instructionPC)]
			ln.Bug = true
			dev.sourceLock.Unlock()
		}
	}
}

// BorrowYieldState will lock the illegal access log for the duration of the
// supplied fucntion, which will be executed with the illegal access log as an
// argument.
func (dev *Developer) BorrowYieldState(f func(*YieldState)) {
	dev.yieldStateLock.Lock()
	defer dev.yieldStateLock.Unlock()
	f(&dev.yieldState)
}
