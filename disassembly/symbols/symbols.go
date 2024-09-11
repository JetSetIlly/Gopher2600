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
	"slices"
	"sync"

	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// Symbols contains the all currently defined symbols.
type Symbols struct {
	crit sync.Mutex

	// the master table is made up of three sub-tables
	label []*table
	read  *table
	write *table

	readCommands      *commandline.Commands
	writeCommands     *commandline.Commands
	readwriteCommands *commandline.Commands
	labelCommands     *commandline.Commands
}

// newSymbols is the preferred method of initialisation for the Symbols type. In
// many instances however, ReadSymbolsFile() might be more appropriate.
func (sym *Symbols) initialise(numBanks int) {
	// AfterLabelChange() and AfterSymbolChange() both acquire the critical lock
	// so we defer them before locking/unlocking in this function
	defer sym.AfterLabelChange()
	defer sym.AfterSymbolChange()

	sym.crit.Lock()
	defer sym.crit.Unlock()

	sym.label = make([]*table, numBanks)
	for i := range sym.label {
		sym.label[i] = newTable()
	}

	sym.read = newTable()
	sym.write = newTable()

	sym.canonise(nil)
}

// should be called in critical section
func (sym *Symbols) resort() {
	for _, l := range sym.label {
		l.sort()
	}
	sym.read.sort()
	sym.write.sort()
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

// Get symbol from label table.
//
// The problem with this function is that it can't handle getting labels if a
// JMP for example, triggers a bankswtich at the same time. We can see this in
// E7 type cartridges. For example the second instruction of HeMan JMPs from
// bank 7 to bank 5. Short of having a copy of the label in every bank, or more
// entwined knowledge of how cartridge mappers work, there's not a lot we can
// do about this.
func (sym *Symbols) GetLabel(bank int, addr uint16) (Entry, bool) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if bank >= len(sym.label) {
		return Entry{}, false
	}

	addr, _ = memorymap.MapAddress(addr, true)

	if e, ok := sym.label[bank].byAddr[addr]; ok {
		return e, ok
	}

	return Entry{}, false
}

// GetReadSymbol from symbols table.
func (sym *Symbols) GetReadSymbol(val uint16, allowSystemSymbols bool) (Entry, bool) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	// we allow address mapping of system symbols
	ma, _ := memorymap.MapAddress(val, true)
	e, ok := sym.read.get(ma)
	if !ok {
		return e, false
	}
	if e.Source == SourceSystem {
		return e, allowSystemSymbols
	}

	return sym.read.get(val)
}

// GetWriteSymbol from symbols table.
func (sym *Symbols) GetWriteSymbol(val uint16) (Entry, bool) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	// we allow address mapping of system symbols
	ma, _ := memorymap.MapAddress(val, false)
	e, ok := sym.write.get(ma)
	if !ok {
		return e, false
	}
	if e.Source == SourceSystem {
		return e, true
	}

	return sym.write.get(val)
}

// SymbolSource identifies the source of the symbol.
type SymbolSource string

// List of valid SymbolSource values.
const (
	SourceDASM      SymbolSource = "DASM"
	SourceAuto      SymbolSource = "Auto"
	SourceSystem    SymbolSource = "System"
	SourceCartridge SymbolSource = "Cartridge"
	SourceCustom    SymbolSource = "Custom"
)

// Add symbol to label table. Symbol will be modified so that it is unique in
// the label table.
func (sym *Symbols) AddLabel(source SymbolSource, bank int, addr uint16, symbol string) bool {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if bank >= len(sym.label) {
		return false
	}

	addr, _ = memorymap.MapAddress(addr, true)

	return sym.label[bank].add(source, addr, symbol)
}

// Add symbol to label table using a symbols created from the address information.
func (sym *Symbols) AddLabelAuto(bank int, addr uint16) bool {
	return sym.AddLabel(SourceAuto, bank, addr, fmt.Sprintf("L%04X", addr))
}

// Remove label from label table. Symbol will be modified so that it is unique
// in the label table.
func (sym *Symbols) RemoveLabel(source SymbolSource, bank int, addr uint16) bool {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	addr, _ = memorymap.MapAddress(addr, true)

	if sym.label[bank].byAddr[addr].Source != source {
		return false
	}

	return sym.label[bank].remove(addr)
}

// Update symbol in label table. Returns success.
func (sym *Symbols) UpdateLabel(source SymbolSource, bank int, addr uint16, oldLabel string, newLabel string) bool {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if bank >= len(sym.label) {
		return false
	}

	addr, _ = memorymap.MapAddress(addr, true)

	return sym.label[bank].update(source, addr, oldLabel, newLabel)
}

// AddSymbol to read/write table. Symbol will be modified so that it is unique
// in the selected table.
//
// The read argument selects the table: true -> read table, false -> write table.
func (sym *Symbols) AddSymbol(source SymbolSource, addr uint16, symbol string, read bool) bool {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if source == SourceSystem {
		addr, _ = memorymap.MapAddress(addr, true)
	}

	if read {
		return sym.read.add(source, addr, symbol)
	}

	return sym.write.add(source, addr, symbol)
}

// RemoveSymbol from read/write table.
//
// The read argument selects the table: true -> read table, false -> write table.
func (sym *Symbols) RemoveSymbol(source SymbolSource, addr uint16, read bool) bool {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if source == SourceSystem {
		addr, _ = memorymap.MapAddress(addr, true)
	}

	if read {
		return sym.read.remove(addr)
	}

	return sym.write.remove(addr)
}

// UpdateSymbol in read/write table. Symbol will be modified so that it is
// unique in the selected table
//
// The read argument selects the table: true -> read table, false -> write table.
func (sym *Symbols) UpdateSymbol(source SymbolSource, addr uint16, oldLabel string, newLabel string, read bool) bool {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	if source == SourceSystem {
		addr, _ = memorymap.MapAddress(addr, true)
	}

	if read {
		return sym.read.update(source, addr, oldLabel, newLabel)
	}

	return sym.write.update(source, addr, oldLabel, newLabel)
}

// AfterLabelChange should be run whenever a label has been added, removed or
// updated. It is okay to batch many changes between calls to AfterLabelChange()
func (sym *Symbols) AfterLabelChange() {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	var s []string
	var err error

	for _, l := range sym.label {
		for _, e := range l.byAddr {
			s = append(s, e.Symbol)
		}
	}

	slices.Sort(s)
	s = slices.Compact(s)

	sym.labelCommands, err = commandline.ParseCommandTemplate(s)
	if err != nil {
		logger.Logf(logger.Allow, "symbols", "creating commandline.Extension: %s", err.Error())
	}
}

// AfterSymbolChange should be run whenever a symbo has been added, removed or
// updated. It is okay to batch many changes between calls to AfterSymbolChange()
func (sym *Symbols) AfterSymbolChange() {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	var read []string
	var write []string
	var err error

	// read symbols
	for _, e := range sym.read.byAddr {
		read = append(read, e.Symbol)
	}

	slices.Sort(read)
	read = slices.Compact(read)

	sym.readCommands, err = commandline.ParseCommandTemplate(read)
	if err != nil {
		logger.Logf(logger.Allow, "symbols", "creating commandline.Extension: %s", err.Error())
	}

	// write symbols
	for _, e := range sym.write.byAddr {
		write = append(write, e.Symbol)
	}

	slices.Sort(write)
	write = slices.Compact(write)

	sym.readCommands, err = commandline.ParseCommandTemplate(write)
	if err != nil {
		logger.Logf(logger.Allow, "symbols", "creating commandline.Extension: %s", err.Error())
	}

	// all symbols
	all := append(read, write...)

	slices.Sort(all)
	all = slices.Compact(all)

	sym.readwriteCommands, err = commandline.ParseCommandTemplate(all)
	if err != nil {
		logger.Logf(logger.Allow, "symbols", "creating commandline.Extension: %s", err.Error())
	}
}

// CommandExtension implements commandline.Extension interface
func (sym *Symbols) CommandExtension(extension string) *commandline.Commands {
	switch extension {
	case "read symbol":
		return sym.readCommands
	case "write symbol":
		return sym.writeCommands
	case "symbol":
		return sym.readwriteCommands
	case "label":
		return sym.labelCommands
	}
	return nil
}
