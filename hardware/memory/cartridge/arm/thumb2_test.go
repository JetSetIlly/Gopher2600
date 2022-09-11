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

package arm_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/test"
)

func TestIT(t *testing.T) {
	arm, mem := prepareTestARM()

	// --- first run
	arm.ClearCaches()

	// MOV R0, #fe
	mem.data[1] = 0b00100000
	mem.data[0] = 0b11111110

	mem.SoftwareInterrupt(2)
	arm.Run()
	test.Equate(t, arm.String(), `R0 : 000000fe		R1 : 00000000		R2 : 00000000		R3 : 00000000
R4 : 00000000		R5 : 00000000		R6 : 00000000		R7 : 00000000
R8 : 00000000		R9 : 00000000		R10: 00000000		R11: 00000000
R12: 00000000		R13: 000003ff		R14: 00000000		R15: 00000006`)
	test.Equate(t, arm.Status().String(), "nzvc   itMask: 0000")

	// --- second run
	arm.ClearCaches()

	// MOV R0, #fe

	// IT neq
	mem.data[3] = 0b10111111
	mem.data[2] = 0b00011000

	// last MOV operation was non-zero so neq comparison will be true

	// MOV R0, #f0
	mem.data[5] = 0b00100000
	mem.data[4] = 0b11110000

	mem.SoftwareInterrupt(6)
	arm.Run()
	test.Equate(t, arm.String(), `R0 : 000000f0		R1 : 00000000		R2 : 00000000		R3 : 00000000
R4 : 00000000		R5 : 00000000		R6 : 00000000		R7 : 00000000
R8 : 00000000		R9 : 00000000		R10: 00000000		R11: 00000000
R12: 00000000		R13: 000003ff		R14: 00000000		R15: 0000000a`)
	test.Equate(t, arm.Status().String(), "nzvc   itMask: 0000")

	// --- third run
	arm.ClearCaches()

	// MOV R0, #fe

	// IT eq
	mem.data[3] = 0b10111111
	mem.data[2] = 0b00001000

	// last MOV operation was non-zero so neq comparison will be false skipping
	// the next instruction

	// MOV R0, #f0

	arm.Run()
	test.Equate(t, arm.String(), `R0 : 000000fe		R1 : 00000000		R2 : 00000000		R3 : 00000000
R4 : 00000000		R5 : 00000000		R6 : 00000000		R7 : 00000000
R8 : 00000000		R9 : 00000000		R10: 00000000		R11: 00000000
R12: 00000000		R13: 000003ff		R14: 00000000		R15: 0000000a`)
	test.Equate(t, arm.Status().String(), "nzvc   itMask: 0000")

	// --- fourth run
	arm.ClearCaches()

	// MOV R0, #fe

	// ITE eq
	mem.data[3] = 0b10111111
	mem.data[2] = 0b00001100

	// last MOV operation was non-zero so neq comparison will be false skipping
	// the next instruction

	// MOV R0, #f0

	// still in IT block but an ELSE command in the mask, meaning that the next
	// instruction will run

	// MOV R0, #00
	mem.data[7] = 0b00100000
	mem.data[6] = 0b00000000

	mem.SoftwareInterrupt(8)
	arm.Run()
	test.Equate(t, arm.String(), `R0 : 00000000		R1 : 00000000		R2 : 00000000		R3 : 00000000
R4 : 00000000		R5 : 00000000		R6 : 00000000		R7 : 00000000
R8 : 00000000		R9 : 00000000		R10: 00000000		R11: 00000000
R12: 00000000		R13: 000003ff		R14: 00000000		R15: 0000000c`)

	// a MOV of zero would mean the status flag being updated, however because
	// we're in a IT block the status flags are not updated
	test.Equate(t, arm.Status().String(), "nzvc   itMask: 0000")
}
