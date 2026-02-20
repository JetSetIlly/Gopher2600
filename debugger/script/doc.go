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

// Package script allows the debugger to record and replay batch scripts.
//
// Scripts can of course be handwritten and be run as though they had been
// written by the debugger. In this instance however, there is a risk that there
// will be errors.
//
// In the case of errors, invalid commands should not be written to the script
// file by the Write type.
//
// Scripts can be run while writing a new script. The action of running the
// script will be recorded in the new script.
//
// Package script is also for input from terminals to make sure the input is
// normalised. (ie. comments ignored; lines split correctly; etc.)
package script
