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

// Package execution tracks the result of instruction execution on the CPU.
// The Result type stores detailed information about each instruction
// encountered during a program's execution on the CPU. A Result can then be
// used to produce output for disassemblers and debuggers with the help of the
// disassembly package.
//
// The Result.IsValid() function can be used to check whether results are
// consistent with the instruction definition. The CPU package doesn't call
// this function because it would introduce unwanted performance penalties, but
// it's probably okay to use in a debugging context.
package execution
