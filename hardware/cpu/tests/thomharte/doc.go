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

// Package thomaharte contains 6502 single-step tests as created/maintained by
// Thom Harte.
//
// https://github.com/SingleStepTests/65x02
//
// The tests are large and are not included as part of the Gopher2600
// repository. In fact, they are excluded by the project's .gitignore file.
//
// Add the instructions you want to test from the 6502/v1 directory on Github to
// the 6502/v1 directory in this package.
//
// The full test suite is slow so by default no tests will run. To enable the
// individual opcode tests set the GOPHER2600_SINGLESTEP_TEST environment
// variable. For example will run the tests for every opcode.
//
//	GOPHER2600_SINGLESTEP_TEST=00-ff go test -test.v .
//
// Opcodes can be specified individually and separated by a comma.
//
//  00,12,3d,fd
//
// Or as a range or as a mixture of both.
//
//  00-0f,23,45,a4-a9
//
// Currently there are some opcode tests that don't succeed with the current
// implementation - there is something about those instructions which I don't
// understand. When those opcodes appear as part of a range they are skipped.
// When they appear singly, there are forcibly run. The log output indicates
// that.
//
// The current set of (working but not single-step-passing opcodes) is:
//
//	0x93	AHX (indirect inexed)
//	0x9f	AHX (absolute indexed Y)
//	0x9b	TAS
//	0x9c	SHY
//	0x9e	SHX

package thomharte
