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

package coprocessor

import (
	"github.com/jetsetilly/gopher2600/coprocessor/mapfile"
	"github.com/jetsetilly/gopher2600/coprocessor/objdump"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
)

// Developer implements the CartCoProcDeveloper interface.
type Developer struct {
	cart mapper.CartCoProcBus

	// mapfile for binary (if available)
	mapfile *mapfile.Mapfile

	// obj dump for binary (if available)
	Source *objdump.ObjDump
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

	dev.mapfile, err = mapfile.NewMapFile(pathToROM)
	if err != nil {
		logger.Logf("developer", err.Error())
	}

	dev.Source, err = objdump.NewObjDump(pathToROM)
	if err != nil {
		logger.Logf("developer", err.Error())
	}

	return dev
}

// LookupSource implements the CartCoProcDeveloper interface.
func (dev *Developer) LookupSource(addr uint32) {
	if dev.mapfile != nil {
		programLabel := dev.mapfile.FindProgramAccess(addr)
		if programLabel != "" {
			logger.Logf("developer", "mapfile: %s()", programLabel)
		}
	}

	if dev.Source != nil {
		src := dev.Source.FindProgramAccess(addr)
		if src != "" {
			logger.Logf("developer", "objdump: %s", src)
		}

	}
}
