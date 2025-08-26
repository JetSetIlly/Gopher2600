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

package dwarf

import (
	"debug/dwarf"
	"debug/elf"
	"encoding/binary"
	"path/filepath"

	"github.com/jetsetilly/gopher2600/coprocessor"
)

// elfShim is a shim for a real elf.File. it implements the coprocessor.CartCoProcELF interface
// which is all that's really required for the gopher2600 dwarf package
type elfShim struct {
	ef *elf.File
}

func (ef *elfShim) Section(name string) ([]uint8, uint32) {
	sec := ef.ef.Section(name)
	if sec == nil {
		return []uint8{}, 0
	}
	d, _ := sec.Data()
	return d, uint32(sec.Addr)
}

func (ef *elfShim) ExecutableSections() []string {
	var x []string
	for _, s := range ef.ef.Sections {
		if s.Flags&elf.SHF_EXECINSTR == elf.SHF_EXECINSTR {
			x = append(x, s.Name)
		}
	}
	return x
}

func (ef *elfShim) DWARF() (*dwarf.Data, error) {
	return ef.ef.DWARF()
}

func (ef *elfShim) ByteOrder() binary.ByteOrder {
	return ef.ef.ByteOrder
}

func (ef *elfShim) Symbols() []elf.Symbol {
	syms, _ := ef.ef.Symbols()
	return syms
}

func (ef *elfShim) PXE() (bool, uint32) {
	return false, 0
}

// find the corresponding ELF file for the specified rom file. the ELF file may
// or may not have DWARF data, the DWARF() function will return an error if not
func findELF(romFile string) coprocessor.CartCoProcELF {
	// try the ROM file itself. it might be an ELF file
	ef, err := elf.Open(romFile)
	if err == nil {
		return &elfShim{ef: ef}
	}

	// the file is not an ELF file so the remainder of the function will work
	// with the path component of the ROM file only
	pathToROM := filepath.Dir(romFile)

	filenames := []string{
		"armcode.elf",
		"custom2.elf",
		"main.elf",
		"ACE_debugging.elf",
	}

	subpaths := []string{
		"",
		"main",
		filepath.Join("main", "bin"),
		filepath.Join("custom", "bin"),
		"arm",
	}

	for _, p := range subpaths {
		for _, f := range filenames {
			ef, err = elf.Open(filepath.Join(pathToROM, p, f))
			if err == nil {
				return &elfShim{ef: ef}
			}
		}
	}

	return nil
}
