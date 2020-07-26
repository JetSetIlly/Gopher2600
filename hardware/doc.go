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

// Package hardware is the base package for the VCS emulation. It and its
// sub-package contain everything required for a headless emulation.
//
// The VCS type is the root of the emulation and contains external references
// to all the VCS sub-systems. From here, the emulation can either be started
// to run continuously (with optional callback to check for continuation); or
// it can be stepped cycle by cycle. Both CPU and video cycle stepping are
// supported.
package hardware
