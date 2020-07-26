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

// Package script allows the debugger to record and replay debugging scripts.
// In this package we refer to this as scribing and rescribing.
//
// Scripts can of course be handwritten and be rescribed as though they had
// been scribed by the debugger. In this instance however, there is a risk that
// there will be errors - invalid commands will not be written to the script
// file by the Scribe type. On Rescribing, invalid commands will attempt to be
// replayed and the appropriate error message printed to the terminal. Comment
// lines begin with the # symbol.
//
// Scripts can be run when scribing a new script. The action of running the
// script will be recorded in the new script.
//
// The Rescribe type satisfies the terminal.Input and is used as a source for
// the debugger packages input loop.
package script
