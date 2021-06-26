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

package symbols

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// put canonical symbols into table. prefer flag should be true if canonical
// names are to supercede any existing symbol.
//
// should be called in critical section.
func (sym *Symbols) canonise(cart *cartridge.Cartridge) {
	defer sym.resort()

	// loop through the array of canonical names.
	//
	// note that because Read and Write in the addresses package are sparse
	// arrays we need to filter out the empty entries. (the Read and Write
	// structures used to be maps and we didn't need to do this)
	for k, v := range addresses.ReadSymbols {
		sym.read.add(SourceSystem, k, v)
	}
	for k, v := range addresses.WriteSymbols {
		sym.write.add(SourceSystem, k, v)
	}

	// add cartridge canonical symbols from cartridge hotspot information
	if cart == nil {
		return
	}

	hb := cart.GetCartHotspots()
	if hb == nil {
		return
	}

	for k, v := range hb.ReadHotspots() {
		ma, area := memorymap.MapAddress(k, true)
		if area != memorymap.Cartridge {
			logger.Logf("symbols", "%s reporting hotspot (%s) outside of cartridge address space", cart.ID(), v.Symbol)
		}
		sym.read.add(SourceCartridge, ma, v.Symbol)
	}

	for k, v := range hb.WriteHotspots() {
		ma, area := memorymap.MapAddress(k, false)
		if area != memorymap.Cartridge {
			logger.Logf("symbols", "%s reporting hotspot (%s) outside of cartridge address space", cart.ID(), v.Symbol)
		}
		sym.write.add(SourceCartridge, ma, v.Symbol)
	}
}
