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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// Package addresses countains all information about VCS addresses, including
// canonical symbols for read and write addresses. These symbols are used by
// the symbols package to create the symbol table for an inserted cartridge.
// They will be supplemented by cartridge specific symbols if a symbols file is
// available (see symbols package for details).
//
// In addition to the canonical maps, there are two sparse arrays Read and
// Write, created from the canonical maps at run time. These arrays are used by
// the emulator for speed purposes - accessing a map although very convnient,
// is noticeably slower than accessing a sparse array. There is probably no
// need to use this arrays outside of the emulation code.
//
// "TIA Registers" and "RIOT Registers" are so named because to those areas,
// those addresses look like registers. They probably don't need referring to
// outside the emulation code.
//
// DataMasks help implement VCS data/address bus artefacts (fully explained
// beloew) and probably don't need to be referred to outside the emulation
// code.
package addresses
