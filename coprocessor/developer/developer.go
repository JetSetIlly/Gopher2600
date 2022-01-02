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

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

// Developer implements the CartCoProcDeveloper interface.
type Developer struct {
	cart mapper.CartCoProcBus

	// mapfile for binary (if available)
	mapfile *mapfile

	// obj dump for binary (if available)
	source     *Source
	sourceLock sync.Mutex
}

// NewDeveloper is the preferred method of initialisation for the Developer type.
func NewDeveloper(pathToROM string, cart mapper.CartCoProcBus) *Developer {
	if cart == nil {
		return nil
	}

	var err error

	dev := &Developer{
		cart: cart,
	}

	dev.cart.SetDeveloper(dev)

	dev.mapfile, err = newMapFile(pathToROM)
	if err != nil {
		logger.Logf("developer", err.Error())
	}

	dev.source, err = newSource(pathToROM)
	if err != nil {
		logger.Logf("developer", err.Error())
	}

	return dev
}

// LookupSource implements the CartCoProcDeveloper interface.
func (dev *Developer) LookupSource(addr uint32) mapper.CoProcSourceReference {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	var ref mapper.CoProcSourceReference

	if dev.source != nil {
		ref = dev.source.findProgramAccess(addr)
	}

	if dev.mapfile != nil {
		ref.Function = dev.mapfile.findProgramAccess(addr)
	}

	return ref
}

// ExecutionProfile implements the CartCoProcDeveloper interface.
func (dev *Developer) ExecutionProfile(addr map[uint32]float32) {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	if dev.source != nil {
		for k, v := range addr {
			dev.source.execute(k, v)
		}
	}
}

// BorrowSource will lock the source code structure for the durction of the
// supplied function, which will be executed with the source code structure as
// an argument.
//
// Should not be called from the emulation goroutine.
func (dev *Developer) BorrowSource(f func(*Source)) {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()
	f(dev.source)
}
