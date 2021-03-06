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

// Package userinput handles input from real hardware that the the user of the
// emulator is using to control the emualted console.
//
// It can be thought of as a translation layer between the GUI implementation
// and the hardware riot.ports package. As such, this package attempts to hide
// details of the GUI implementation while protecting the ports package from
// complication.
//
// The GUI implementation in use during development was SDL and so there will
// be a bias towards that system.
package userinput
