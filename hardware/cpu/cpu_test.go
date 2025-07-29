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

package cpu_test

import (
	"fmt"
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/cpu"
	rtest "github.com/jetsetilly/gopher2600/hardware/cpu/registers/test"
	"github.com/jetsetilly/gopher2600/test"
)

type testMem struct {
	internal []uint8
}

func newTestMem() *testMem {
	return &testMem{
		internal: make([]uint8, 0x10000),
	}
}

func (mem *testMem) putInstructions(origin uint16, bytes ...uint8) uint16 {
	for i, b := range bytes {
		_ = mem.Write(uint16(i)+origin, b)
	}
	return origin + uint16(len(bytes))
}

func (mem testMem) assert(t *testing.T, address uint16, expected uint8) {
	t.Helper()
	d, _ := mem.Read(address)
	if d != expected {
		t.Errorf("memory assertion failed: %#04x = %#02x (expected %#02x)", address, d, expected)
	}
}

// Clear sets all bytes in memory to zero.
func (mem *testMem) Clear() {
	for i := 0; i < len(mem.internal); i++ {
		mem.internal[i] = 0
	}
}

func (mem *testMem) IsAddressError(_ error) bool {
	// test memory never returns an address error
	return false
}

func (mem testMem) Read(address uint16) (uint8, error) {
	return mem.internal[address], nil
}

func (mem testMem) ReadZeroPage(address uint8) (uint8, error) {
	return mem.Read(uint16(address))
}

func (mem *testMem) Write(address uint16, data uint8) error {
	mem.internal[address] = data
	return nil
}

func step(t *testing.T, mc *cpu.CPU) {
	t.Helper()
	err := mc.ExecuteInstruction(cpu.NilCycleCallback)
	if err != nil {
		fmt.Println(mc.LastResult.Defn)
		t.Fatal(err)
	}
	err = mc.LastResult.IsValid()
	if err != nil {
		t.Fatal(err)
	}
}

func testStatusInstructions(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	// SEC; CLC; CLI; SEI; SED; CLD; CLV
	origin = mem.putInstructions(origin, 0x38, 0x18, 0x58, 0x78, 0xf8, 0xd8, 0xb8)
	step(t, mc) // SEC
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzC")
	step(t, mc) // CLC
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")
	step(t, mc) // CLI
	rtest.EquateRegisters(t, mc.Status, "sv-Bdizc")
	step(t, mc) // SEI
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")
	step(t, mc) // SED
	rtest.EquateRegisters(t, mc.Status, "sv-BDIzc")
	step(t, mc) // CLD
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")
	step(t, mc) // CLV
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")

	// PHP; PLP
	_ = mem.putInstructions(origin, 0x08, 0x28)
	step(t, mc) // PHP
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")
	rtest.EquateRegisters(t, mc.SP.Data, 0xfc)

	// mangle status register
	mc.Status.Sign = true
	mc.Status.Overflow = true
	mc.Status.Break = false

	// restore status register
	step(t, mc) // PLP
	rtest.EquateRegisters(t, mc.SP.Data, 0xfd)

	// break flag is always set on PLP
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")
}

func testRegsiterArithmetic(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	// LDA immediate; ADC immediate
	origin = mem.putInstructions(origin, 0xa9, 1, 0x69, 10)
	step(t, mc) // LDA #1
	step(t, mc) // ADC #10
	rtest.EquateRegisters(t, mc.A, 11)

	// SEC; SBC immediate
	_ = mem.putInstructions(origin, 0x38, 0xe9, 8)
	step(t, mc) // SEC
	step(t, mc) // SBC #8
	rtest.EquateRegisters(t, mc.A, 3)
}

func testRegsiterBitwiseInstructions(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	// ORA immediate; EOR immediate; AND immediate
	origin = mem.putInstructions(origin, 0x09, 0xff, 0x49, 0xf0, 0x29, 0x01)
	rtest.EquateRegisters(t, mc.A, 0)
	step(t, mc) // ORA #$FF
	rtest.EquateRegisters(t, mc.A, 255)
	step(t, mc) // EOR #$F0
	rtest.EquateRegisters(t, mc.A, 15)
	step(t, mc) // AND #$01
	rtest.EquateRegisters(t, mc.A, 1)

	// ASL implied; LSR implied; LSR implied
	origin = mem.putInstructions(origin, 0x0a, 0x4a, 0x4a)
	step(t, mc) // ASL
	rtest.EquateRegisters(t, mc.A, 2)
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")
	step(t, mc) // LSR
	rtest.EquateRegisters(t, mc.A, 1)
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")
	step(t, mc) // LSR
	rtest.EquateRegisters(t, mc.A, 0)
	rtest.EquateRegisters(t, mc.Status, "sv-BdIZC")

	// ROL implied; ROR implied; ROR implied; ROR implied
	_ = mem.putInstructions(origin, 0x2a, 0x6a, 0x6a, 0x6a)
	step(t, mc) // ROL
	rtest.EquateRegisters(t, mc.A, 1)
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")
	step(t, mc) // ROR
	rtest.EquateRegisters(t, mc.A, 0)
	rtest.EquateRegisters(t, mc.Status, "sv-BdIZC")
	step(t, mc) // ROR
	rtest.EquateRegisters(t, mc.A, 128)
	rtest.EquateRegisters(t, mc.Status, "Sv-BdIzc")
	step(t, mc) // ROR
	rtest.EquateRegisters(t, mc.A, 64)
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")
}

func testImmediateImplied(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	// LDX immediate; INX; DEX
	origin = mem.putInstructions(origin, 0xa2, 5, 0xe8, 0xca)
	step(t, mc) // LDX #5
	rtest.EquateRegisters(t, mc.X, 5)
	step(t, mc) // INX
	rtest.EquateRegisters(t, mc.X, 6)
	step(t, mc) // DEX
	rtest.EquateRegisters(t, mc.X, 5)
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")

	// PHA; LDA immediate; PLA
	origin = mem.putInstructions(origin, 0xa9, 5, 0x48, 0xa9, 0, 0x68)
	step(t, mc) // LDA #5
	step(t, mc) // PHA
	rtest.EquateRegisters(t, mc.SP.Data, 0xfc)
	step(t, mc) // LDA #0
	rtest.EquateRegisters(t, mc.A, 0)
	rtest.EquateRegisters(t, mc.Status, "sv-BdIZc")
	step(t, mc) // PLA
	rtest.EquateRegisters(t, mc.A, 5)

	// TAX; TAY; LDX immediate; TXA; LDY immediate; TYA; INY; DEY
	origin = mem.putInstructions(origin, 0xaa, 0xa8, 0xa2, 1, 0x8a, 0xa0, 2, 0x98, 0xc8, 0x88)
	step(t, mc) // TAX
	rtest.EquateRegisters(t, mc.X, 5)
	step(t, mc) // TAY
	rtest.EquateRegisters(t, mc.Y, 5)
	step(t, mc) // LDX #1
	step(t, mc) // TXA
	rtest.EquateRegisters(t, mc.A, 1)
	step(t, mc) // LDY #2
	step(t, mc) // TYA
	rtest.EquateRegisters(t, mc.A, 2)
	step(t, mc) // INY
	rtest.EquateRegisters(t, mc.Y, 3)
	step(t, mc) // DEY
	rtest.EquateRegisters(t, mc.Y, 2)

	// TSX; LDX immediate; TXS
	_ = mem.putInstructions(origin, 0xba, 0xa2, 100, 0x9a)
	step(t, mc) // TSX
	rtest.EquateRegisters(t, mc.X, 0xfd)
	step(t, mc) // LDX #100
	step(t, mc) // TXS
	rtest.EquateRegisters(t, mc.SP.Data, 100)
}

func testOtherAddressingModes(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	mem.putInstructions(0x0100, 123, 43)
	mem.putInstructions(0x01a2, 47)

	// LDA zero page
	origin = mem.putInstructions(origin, 0xa5, 0x00)
	step(t, mc) // LDA $00
	rtest.EquateRegisters(t, mc.A, 0xa5)

	// LDX immediate; LDA zero page,X
	origin = mem.putInstructions(origin, 0xa2, 1, 0xb5, 0x01)
	step(t, mc) // LDX #1
	step(t, mc) // LDA 01,X
	rtest.EquateRegisters(t, mc.A, 0xa2)

	// LDY immediate; LDX zero page,Y
	origin = mem.putInstructions(origin, 0xa0, 3, 0xb6, 0x01)
	step(t, mc) // LDX #3
	step(t, mc) // LDA 01,Y
	rtest.EquateRegisters(t, mc.A, 0xa2)

	// LDA absolute
	origin = mem.putInstructions(origin, 0xad, 0x00, 0x01)
	step(t, mc) // LDA $0100
	rtest.EquateRegisters(t, mc.A, 123)

	// LDX immediate; LDA absolute,X
	origin = mem.putInstructions(origin, 0xa2, 1, 0xbd, 0x01, 0x00)
	step(t, mc) // LDX #1
	rtest.EquateRegisters(t, mc.X, 1)
	step(t, mc) // LDA $0001,X
	rtest.EquateRegisters(t, mc.A, 0xa2)

	// LDY immediate; LDA absolute,Y
	origin = mem.putInstructions(origin, 0xa0, 1, 0xb9, 0x01, 0x00)
	step(t, mc) // LDY #1
	rtest.EquateRegisters(t, mc.X, 1)
	step(t, mc) // LDA $0001,Y
	rtest.EquateRegisters(t, mc.A, 0xa2)

	// pre-indexed indirect
	// X = 1
	// INX; LDA (Indirect, X)
	origin = mem.putInstructions(origin, 0xe8, 0xa1, 0x0b)
	step(t, mc) // INX (x equals 2)
	step(t, mc) // LDA (0x0b,X)

	// post-indexed indirect (see below)

	// pre-indexed indirect (with wraparound)
	// X = 1
	// INX; LDA (Indirect, X)
	origin = mem.putInstructions(origin, 0xe8, 0xa1, 0xff)
	step(t, mc) // INX (x equals 2)
	step(t, mc) // LDA (0xff,X)
	rtest.EquateRegisters(t, mc.A, 47)

	// post-indexed indirect (with page-fault)
	// Y = 1
	// INY; INY; LDA (Indirect), Y
	mem.putInstructions(0xc0, 0xfd, 0x00)
	_ = mem.putInstructions(origin, 0xc8, 0xc8, 0xb1, 0xc0)
	step(t, mc) // INY (y = 2)
	step(t, mc) // INY (y = 2)
	step(t, mc) // LDA (0x0b),Y
	rtest.EquateRegisters(t, mc.A, 123)
	if mc.LastResult.PageFault != true {
		t.Errorf("expected page-fault")
	}
}

func testPostIndexedIndirect(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	mem.putInstructions(0xee00, 0x01, 0x02, 0x03)

	mc.PC.Load(0x04)
	origin = mem.putInstructions(origin, 0x01, 0xee, 0xfe, 0xfd)
	origin = mem.putInstructions(origin, 0xa0, 0x01)
	step(t, mc)
	rtest.EquateRegisters(t, mc.Y, 1)
	_ = mem.putInstructions(origin, 0xb1, 0x00)
	step(t, mc)
	rtest.EquateRegisters(t, mc.A, 0x03)
}

func testStorageInstructions(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	// LDA immediate; STA absolute
	origin = mem.putInstructions(origin, 0xa9, 0x54, 0x8d, 0x00, 0x01)
	step(t, mc) // LDA 0x54
	step(t, mc) // STA 0x0100
	mem.assert(t, 0x0100, 0x54)

	// LDX immediate; STX absolute
	origin = mem.putInstructions(origin, 0xa2, 0x63, 0x8e, 0x01, 0x01)
	step(t, mc) // LDX 0x63
	step(t, mc) // STX 0x0101
	mem.assert(t, 0x0101, 0x63)

	// LDY immediate; STY absolute
	origin = mem.putInstructions(origin, 0xa0, 0x72, 0x8c, 0x02, 0x01)
	step(t, mc) // LDY 0x72
	step(t, mc) // STY 0x0102
	mem.assert(t, 0x0101, 0x63)

	// INC zero page
	origin = mem.putInstructions(origin, 0xe6, 0x01)
	step(t, mc) // INC $01
	mem.assert(t, 0x01, 0x55)

	// DEC absolute
	_ = mem.putInstructions(origin, 0xce, 0x00, 0x01)
	step(t, mc) // DEC 0x0100
	mem.assert(t, 0x0100, 0x53)
}

func testBranching(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	origin = 0
	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = mem.putInstructions(origin, 0x10, 0x10)
	step(t, mc) // BPL $10
	rtest.EquateRegisters(t, mc.PC, 0x12)

	origin = 0
	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = mem.putInstructions(origin, 0x50, 0x10)
	step(t, mc) // BVC $10
	rtest.EquateRegisters(t, mc.PC, 0x12)

	origin = 0
	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = mem.putInstructions(origin, 0x90, 0x10)
	step(t, mc) // BCC $10
	rtest.EquateRegisters(t, mc.PC, 0x12)

	origin = 0
	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = mem.putInstructions(origin, 0x38, 0xb0, 0x10)
	step(t, mc) // SEC
	step(t, mc) // BCS $10
	rtest.EquateRegisters(t, mc.PC, 0x13)

	origin = 0
	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = mem.putInstructions(origin, 0xe8, 0xd0, 0x10)
	step(t, mc) // INX
	step(t, mc) // BNE $10
	rtest.EquateRegisters(t, mc.PC, 0x13)

	origin = 0
	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = mem.putInstructions(origin, 0xca, 0x30, 0x10)
	step(t, mc) // DEX
	step(t, mc) // BMI $10
	rtest.EquateRegisters(t, mc.PC, 0x13)

	_ = mem.putInstructions(0x13, 0xe8, 0xf0, 0x10)
	step(t, mc) // INX
	step(t, mc) // BEQ $10
	rtest.EquateRegisters(t, mc.PC, 0x26)

	origin = 0
	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}
	// fudging overflow test
	mc.Status.Overflow = true
	_ = mem.putInstructions(origin, 0x70, 0x10)
	step(t, mc) // BVS $10
	rtest.EquateRegisters(t, mc.PC, 0x12)
}

func testBranchingBackwards(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	origin = 0x20
	err = mc.LoadPC(0x20)
	test.ExpectSuccess(t, err)

	// BPL backwards
	_ = mem.putInstructions(origin, 0x10, 0xfd)
	step(t, mc) // BPL $FF
	rtest.EquateRegisters(t, mc.PC, 0x1f)

	// BVS backwards
	origin = 0x20
	err = mc.LoadPC(0x20)
	test.ExpectSuccess(t, err)
	mc.Status.Overflow = true
	_ = mem.putInstructions(origin, 0x70, 0xfd)
	step(t, mc) // BVS $FF
	rtest.EquateRegisters(t, mc.PC, 0x1f)
}

func testBranchingPageFaults(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	// BNE backwards - with PC wrap (causing a page fault)
	origin = 0x20
	err = mc.LoadPC(0x20)
	test.ExpectSuccess(t, err)
	mc.Status.Zero = false
	_ = mem.putInstructions(origin, 0xd0, 0x80)
	step(t, mc) // BNE $F0
	rtest.EquateRegisters(t, mc.PC, 0xffa2)

	// pagefault flag should be set
	if !mc.LastResult.PageFault {
		t.Errorf("expected pagefault on branch")
	}

	// number of cycles should be 4 instead of 2
	//  +1 for failed branch test (causing PC to jump)
	//  +1 for page fault
	if mc.LastResult.Cycles != 4 {
		t.Errorf("expected pagefault on branch")
	}
}

func testJumps(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	// JMP absolute
	_ = mem.putInstructions(origin, 0x4c, 0x00, 0x01)
	step(t, mc) // JMP $100
	rtest.EquateRegisters(t, mc.PC, 0x0100)

	// JMP indirect
	origin = 0
	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	mem.putInstructions(0x0050, 0x49, 0x01)
	_ = mem.putInstructions(origin, 0x6c, 0x50, 0x00)
	step(t, mc) // JMP ($50)
	rtest.EquateRegisters(t, mc.PC, 0x0149)

	// JMP indirect (bug)
	origin = 0
	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	mem.putInstructions(0x01FF, 0x03)
	mem.putInstructions(0x0100, 0x00)
	_ = mem.putInstructions(origin, 0x6c, 0xFF, 0x01)
	step(t, mc) // JMP ($0x01FF)
	rtest.EquateRegisters(t, mc.PC, 0x0003)
}

func testComparisonInstructions(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	// CMP immediate (equality)
	origin = mem.putInstructions(origin, 0xc9, 0x00)
	step(t, mc) // CMP $00
	rtest.EquateRegisters(t, mc.Status, "sv-BdIZC")

	// LDA immediate; CMP immediate
	origin = mem.putInstructions(origin, 0xa9, 0xf6, 0xc9, 0x18)
	step(t, mc) // LDA $F6
	step(t, mc) // CMP $10
	rtest.EquateRegisters(t, mc.Status, "Sv-BdIzC")

	// LDX immediate; CMP immediate
	origin = mem.putInstructions(origin, 0xa2, 0xf6, 0xe0, 0x18)
	step(t, mc) // LDX $F6
	step(t, mc) // CMP $10
	rtest.EquateRegisters(t, mc.Status, "Sv-BdIzC")

	// LDY immediate; CMP immediate
	origin = mem.putInstructions(origin, 0xa0, 0xf6, 0xc0, 0x18)
	step(t, mc) // LDY $F6
	step(t, mc) // CMP $10
	rtest.EquateRegisters(t, mc.Status, "Sv-BdIzC")

	// LDA immediate; CMP immediate
	origin = mem.putInstructions(origin, 0xa9, 0x18, 0xc9, 0xf6)
	step(t, mc) // LDA $F6
	step(t, mc) // CMP $10
	rtest.EquateRegisters(t, mc.Status, "sv-BdIzc")

	// BIT zero page
	origin = mem.putInstructions(origin, 0x24, 0x01)
	step(t, mc) // BIT $01
	rtest.EquateRegisters(t, mc.Status, "sv-BdIZc")

	// BIT immediate
	_ = mem.putInstructions(origin, 0x24, 0x01)
	step(t, mc) // BIT $01
	rtest.EquateRegisters(t, mc.Status, "sv-BdIZc")
}

func testSubroutineInstructions(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	// JSR absolute
	_ = mem.putInstructions(origin, 0x20, 0x00, 0x01)
	step(t, mc) // JSR $0100
	rtest.EquateRegisters(t, mc.PC, 0x0100)
	mem.assert(t, 0x01fd, 0x00)
	mem.assert(t, 0x01fc, 0x02)
	rtest.EquateRegisters(t, mc.SP.Data, 0xfb)

	_ = mem.putInstructions(0x100, 0x60)
	step(t, mc) // RTS
	rtest.EquateRegisters(t, mc.PC, 0x0003)
	mem.assert(t, 0x01fd, 0x00)
	mem.assert(t, 0x01fc, 0x02)
	rtest.EquateRegisters(t, mc.SP.Data, 0xfd)
}

func testDecimalMode(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	_ = mem.putInstructions(origin, 0xf8, 0xa9, 0x20, 0x18, 0x69, 0x01, 0x38, 0xe9, 0x01)
	step(t, mc) // SED
	step(t, mc) // LDA #$20
	step(t, mc) // CLC
	step(t, mc) // ADC #01
	step(t, mc) // SEC
	step(t, mc) // SBC #$00
	rtest.EquateRegisters(t, mc.A, 0x20)
}

func testBRK(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	_ = mem.putInstructions(origin, 0x69, 0x01, 0x00)
	step(t, mc) // ADC #$01
	rtest.EquateRegisters(t, mc.PC, 0x02)
	rtest.EquateRegisters(t, mc.A, 0x01)
	step(t, mc) // BRK
	rtest.EquateRegisters(t, mc.PC, 0x00)
}

func testKIL(t *testing.T, mc *cpu.CPU, mem *testMem) {
	var origin uint16
	var err error

	mem.Clear()
	err = mc.Reset(nil)
	if err != nil {
		t.Fatal(err)
	}

	_ = mem.putInstructions(origin, 0x02, 0x69, 0x01)
	step(t, mc) // KIL
	rtest.EquateRegisters(t, mc.PC, 0x01)
	step(t, mc) // ADC #$01
	rtest.EquateRegisters(t, mc.PC, 0x01)
	rtest.EquateRegisters(t, mc.A, 0x00)
}

func TestCPU(t *testing.T) {
	mem := newTestMem()
	mc := cpu.NewCPU(mem)

	testStatusInstructions(t, mc, mem)
	testRegsiterArithmetic(t, mc, mem)
	testRegsiterBitwiseInstructions(t, mc, mem)
	testImmediateImplied(t, mc, mem)
	testOtherAddressingModes(t, mc, mem)
	testPostIndexedIndirect(t, mc, mem)
	testStorageInstructions(t, mc, mem)
	testBranching(t, mc, mem)
	testBranchingBackwards(t, mc, mem)
	testBranchingPageFaults(t, mc, mem)
	testJumps(t, mc, mem)
	testComparisonInstructions(t, mc, mem)
	testSubroutineInstructions(t, mc, mem)
	testDecimalMode(t, mc, mem)
	testBRK(t, mc, mem)
	testKIL(t, mc, mem)
}
