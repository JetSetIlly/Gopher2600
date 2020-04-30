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

package cartridge

// StaticArea defines the operations required for a debugger to access
// non-addressable areas of a cartridge.
//
// Some cartridge mappings (eg. DPC) have data that are not addressable by the
// CPU (using DataFetchers to retreive that data in the case of DPC). For
// debuggers therefore we cannot use the usual Read()/Write() mechanism or even
// the Poke() mechanism.
//
// Address origin is 0x0000 is memtop is equal to StaticSize()-1.
type StaticArea interface {
	StaticRead(addr uint16) (data uint8, err error)
	StaticWrite(addr uint16, data uint8) error
	StaticSize() int
}
