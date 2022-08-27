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

// Package govern defines the types that define the current condition of the
// emulation. The three conditions are Mode, State and Event.
//
// Also defined is the method of requesting a state change from the GUI. Most
// often state change comes from the emulation but in some intances it is
// necessary to instruct the emulation to change mode or state - for example,
// from the GUI as a result of the a user request.
package govern
