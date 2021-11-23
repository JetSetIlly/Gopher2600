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

// Package userinput translates real hardware input from the user's computer,
// to the emulated console. It abstracts real user interface to the emulated
// interface.
//
// The Controllers type processes userinput Events (input from a gamepad for
// example) with the HandleUserInput function. That input is translated into
// an event understood by the emulation (paddle left for example).
//
// Many emulations can be attached to a single Controllers type with the
// AddInputHandler(). However, the DrivenEvent mechanism in the riot.ports
// package is a better way to do this in many instances.
package userinput
