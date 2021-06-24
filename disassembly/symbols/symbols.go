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
	"sync"

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// Symbols contains the all currently defined symbols.
type Symbols struct {
	// the master table is made up of three sub-tables
	label []*table
	read  *table
	write *table

	crit sync.Mutex
}

// newSymbols is the preferred method of initialisation for the Symbols type. In
// many instances however, ReadSymbolsFile() might be more appropriate.
func (sym *Symbols) initialise(numBanks int) {
	sym.label = make([]*table, numBanks)
	for i := range sym.label {
		sym.label[i] = newTable()
	}

	sym.read = newTable()
	sym.write = newTable()

	sym.canonise(nil)
}

// LabelWidth returns the maximum number of characters required by a label in
// the label table.
func (sym *Symbols) LabelWidth() int {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	max := 0
	for _, l := range sym.label {
		if l.maxWidth > max {
			max = l.maxWidth
		}
	}
	return max
}

// SymbolWidth returns the maximum number of characters required by a symbol in
// the read/write table.
func (sym *Symbols) SymbolWidth() int {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if sym.read.maxWidth > sym.write.maxWidth {
		return sym.read.maxWidth
	}
	return sym.write.maxWidth
}

// put canonical symbols into table. prefer flag should be true if canonical
// names are to supercede any existing symbol.
//
// should be called in critical section.
func (sym *Symbols) canonise(cart *cartridge.Cartridge) {
	defer sym.reSort()

	// loop through the array of canonical names.
	//
	// note that because Read and Write in the addresses package are sparse
	// arrays we need to filter out the empty entries. (the Read and Write
	// structures used to be maps and we didn't need to do this)
	for k, v := range addresses.ReadSymbols {
		sym.read.add(k, v, true)
	}
	for k, v := range addresses.WriteSymbols {
		sym.write.add(k, v, true)
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
		sym.read.add(ma, v.Symbol, true)
	}

	for k, v := range hb.WriteHotspots() {
		ma, area := memorymap.MapAddress(k, false)
		if area != memorymap.Cartridge {
			logger.Logf("symbols", "%s reporting hotspot (%s) outside of cartridge address space", cart.ID(), v.Symbol)
		}
		sym.write.add(ma, v.Symbol, true)
	}
}

// reSort() should be called whenever any of the sub-tables have been updated.
//
// should be called in critical section.
func (sym *Symbols) reSort() {
	for _, l := range sym.label {
		sort.Sort(l)
	}
	sort.Sort(sym.read)
	sort.Sort(sym.write)
}

// Add symbol to label table using a symbols created from the address information.
func (sym *Symbols) AddLabelAuto(bank int, addr uint16) {
	sym.AddLabel(bank, addr, fmt.Sprintf("L%04X", addr), false)
}

func (sym *Symbols) RemoveLabel(bank int, addr uint16) bool {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	return sym.label[bank].remove(addr)
}

// Add symbol to label table.
func (sym *Symbols) AddLabel(bank int, addr uint16, symbol string, prefer bool) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if bank < len(sym.label) {
		sym.label[bank].add(addr, symbol, prefer)
	}
}

// Get symbol from label table.
//
// The problem with this function is that it can't handle getting labels if a
// JMP for example, triggers a bankswtich at the same time. We can see this in
// E7 type cartridges. For example the second instruction of HeMan JMPs from
// bank 7 to bank 5. Short of having a copy of the label in every bank, or more
// entwined knowledge of how cartridge mappers work, there's not a lot we can
// do about this.
func (sym *Symbols) GetLabel(bank int, addr uint16) (string, bool) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if bank < len(sym.label) {
		if v, ok := sym.label[bank].entries[addr]; ok {
			return v, ok
		}

		// no entry found so try the mapped address
		addr, _ = memorymap.MapAddress(addr, true)
		if v, ok := sym.label[bank].entries[addr]; ok {
			return v, ok
		}
	}
	return "", false
}

// Update symbol in label table. Returns success.
func (sym *Symbols) UpdateLabel(bank int, addr uint16, oldLabel string, newLabel string) bool {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if bank >= len(sym.label) {
		return false
	}

	if s, ok := sym.label[bank].entries[addr]; ok {
		if s == oldLabel {
			sym.label[bank].entries[addr] = newLabel
			return true
		}
	}

	addr, _ = memorymap.MapAddress(addr, true)
	if s, ok := sym.label[bank].entries[addr]; ok {
		if s == oldLabel {
			sym.label[bank].entries[addr] = newLabel
			return true
		}
	}

	return false
}

// Get symbol from read table.
func (sym *Symbols) GetReadSymbol(addr uint16) (string, bool) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if v, ok := sym.read.entries[addr]; ok {
		return v, ok
	}

	// no entry found so try the mapped address
	addr, _ = memorymap.MapAddress(addr, true)
	if v, ok := sym.read.entries[addr]; ok {
		return v, ok
	}

	return "", false
}

// Get symbol from read table.
func (sym *Symbols) GetWriteSymbol(addr uint16) (string, bool) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if v, ok := sym.write.entries[addr]; ok {
		return v, ok
	}

	// no entry found so try the mapped address
	addr, _ = memorymap.MapAddress(addr, false)
	if v, ok := sym.read.entries[addr]; ok {
		return v, ok
	}

	return "", false
}
