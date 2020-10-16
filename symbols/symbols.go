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
	"fmt"
	"sort"

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// Table is the master symbols table for the loaded programme.
type Symbols struct {
	// the master table is made up of three sub-tables
	Label *Table
	Read  *Table
	Write *Table
}

// NewSymbols is the preferred method of initialisation for the Symbols type. In
// many instances however, ReadSymbolsFile() might be more appropriate.
func NewSymbols() *Symbols {
	sym := &Symbols{
		Label: newTable(),
		Read:  newTable(),
		Write: newTable(),
	}
	sym.canonise(nil)
	return sym
}

func (sym *Symbols) LabelWidth() int {
	return sym.Label.maxWidth
}

func (sym *Symbols) SymbolWidth() int {
	if sym.Read.maxWidth > sym.Write.maxWidth {
		return sym.Read.maxWidth
	}
	return sym.Write.maxWidth
}

// put canonical symbols into table. prefer flag should be true if canonical
// names are to supercede any existing symbol.
func (sym *Symbols) canonise(cart *cartridge.Cartridge) {
	defer sym.reSort()

	// loop through the array of canonical names.
	//
	// note that because Read and Write in the addresses package are sparse
	// arrays we need to filter out the empty entries. (the Read and Write
	// structures used to be maps and we didn't need to do this)
	for k, v := range addresses.ReadSymbols {
		sym.Read.add(k, v, true)
	}
	for k, v := range addresses.WriteSymbols {
		sym.Write.add(k, v, true)
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
			logger.Log("symbols", fmt.Sprintf("%s reporting hotspot (%s) outside of cartridge address space", cart.ID(), v.Symbol))
		}
		sym.Read.add(ma, v.Symbol, true)
	}

	for k, v := range hb.WriteHotspots() {
		ma, area := memorymap.MapAddress(k, false)
		if area != memorymap.Cartridge {
			logger.Log("symbols", fmt.Sprintf("%s reporting hotspot (%s) outside of cartridge address space", cart.ID(), v.Symbol))
		}
		sym.Write.add(ma, v.Symbol, true)
	}
}

// reSort() should be called whenever any of the sub-tables have been updated.
func (sym *Symbols) reSort() {
	sort.Sort(sym.Label)
	sort.Sort(sym.Read)
	sort.Sort(sym.Write)
}
