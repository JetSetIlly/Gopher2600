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

// Package setup is used to preset the emulation depending on the attached
// cartridge. It is currently quite limited but is useful none-the-less.
// Currently support entry types:
//
//	Toggling of panel switches
//	Apply patches to cartridge
//	Television specification
//
// Menu driven selection of patches would be a nice feature to have in the
// future. But at the moment, the package doesn't even facilitate editing of
// entries. Adding new entries to the setup database therefore requires editing
// the DB file by hand. For reference the following describes the format of
// each entry type:
//
//	Panel Toggles
//
//	<DB Key>, panel, <SHA-1 Hash>, <player 0 (bool)>, .<player 1 (bool)>, <color (bool)>, <notes>
//
// When editing the DB file, make sure the DB Key is unique
//
//	Patch Cartridge
//
//	<DB Key>, patch, <SHA-1 Hash>, <patch file>, <notes>
//
// Patch files are located in the patches sub-directory of the resources path.
//
//	Television
//
//	<DB Key>, television, <SHA-1 Hash>, <tv spec>, notes
//
// TV spec should be one of PAL or NTSC (or AUTO)
package setup
