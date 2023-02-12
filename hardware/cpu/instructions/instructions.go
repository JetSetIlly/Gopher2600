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

package instructions

import "fmt"

// AddressingMode describes the method data for the instruction should be received.
type AddressingMode int

func (m AddressingMode) String() string {
	switch m {
	case Implied:
		return "Implied"
	case Immediate:
		return "Immediate"
	case Relative:
		return "Relative"
	case Absolute:
		return "Absolute"
	case ZeroPage:
		return "ZeroPage"
	case Indirect:
		return "Indirect"
	case IndexedIndirect:
		return "IndexedIndirect"
	case IndirectIndexed:
		return "IndirectIndexed"
	case AbsoluteIndexedX:
		return "AbsoluteIndexedX"
	case AbsoluteIndexedY:
		return "AbsoluteIndexedY"
	case ZeroPageIndexedX:
		return "ZeroPageIndexedX"
	case ZeroPageIndexedY:
		return "ZeroPageIndexedY"
	}

	return "unknown addressing mode"
}

// List of supported addressing modes.
const (
	Implied AddressingMode = iota
	Immediate
	Relative // relative addressing is used for branch instructions

	Absolute // abs
	ZeroPage // zpg
	Indirect // ind

	IndexedIndirect // (ind,X)
	IndirectIndexed // (ind), Y

	AbsoluteIndexedX // abs,X
	AbsoluteIndexedY // abs,Y

	ZeroPageIndexedX // zpg,X
	ZeroPageIndexedY // zpg,Y
)

// EffectCategory categorises an instruction by the effect it has.
type EffectCategory int

// List of effect categories.
const (
	Read EffectCategory = iota
	Write
	RMW

	// the following three effects have a variable effect on the program
	// counter, depending on the instruction's precise operand.

	// flow consists of the Branch and JMP instructions. Branch instructions
	// specifically can be distinguished by the AddressingMode.
	Flow

	Subroutine
	Interrupt
)

// Cycles is the number of cycles for the instruction. The Formatted value is
// the Value field formatted as a string with the condition that branch instructions
// are formatted as:
//
//	Value/Value+1
//
// We do not format any potential PageSensitive cycle.
type Cycles struct {
	Value     int
	Formatted string
}

// Operator defines which operation is performed by the opcode. Many opcodes
// can perform the same operation.
type Operator int

// List of valid Operator values.
const (
	Nop Operator = iota
	Adc
	AHX
	ANC
	And
	ARR
	Asl
	ASR
	AXS
	Bcc
	Bcs
	Beq
	Bit
	Bmi
	Bne
	Bpl
	Brk
	Bvc
	Bvs
	Clc
	Cld
	Cli
	Clv
	Cmp
	Cpx
	Cpy
	DCP
	Dec
	Dex
	Dey
	Eor
	Inc
	Inx
	Iny
	ISC
	Jmp
	Jsr
	KIL
	LAS
	LAX
	Lda
	Ldx
	Ldy
	Lsr
	NOP
	Ora
	Pha
	Php
	Pla
	Plp
	RLA
	Rol
	Ror
	RRA
	Rti
	Rts
	SAX
	Sbc
	SBC
	Sec
	Sed
	Sei
	SHX
	SHY
	SLO
	SRE
	Sta
	Stx
	Sty
	TAS
	Tax
	Tay
	Tsx
	Txa
	Txs
	Tya
	XAA
)

func (operator Operator) String() string {
	switch operator {
	case Nop:
		return "nop"
	case Adc:
		return "adc"
	case AHX:
		return "AHX"
	case ANC:
		return "ANC"
	case And:
		return "and"
	case ARR:
		return "ARR"
	case Asl:
		return "asl"
	case ASR:
		return "ASR"
	case AXS:
		return "AXS"
	case Bcc:
		return "bcc"
	case Bcs:
		return "bcs"
	case Beq:
		return "beq"
	case Bit:
		return "bit"
	case Bmi:
		return "bmi"
	case Bne:
		return "bne"
	case Bpl:
		return "bpl"
	case Brk:
		return "brk"
	case Bvc:
		return "bvc"
	case Bvs:
		return "bvs"
	case Clc:
		return "clc"
	case Cld:
		return "cld"
	case Cli:
		return "cli"
	case Clv:
		return "clv"
	case Cmp:
		return "cmp"
	case Cpx:
		return "cpx"
	case Cpy:
		return "cpy"
	case DCP:
		return "DCP"
	case Dec:
		return "dec"
	case Dex:
		return "dex"
	case Dey:
		return "dey"
	case Eor:
		return "eor"
	case Inc:
		return "inc"
	case Inx:
		return "inx"
	case Iny:
		return "iny"
	case ISC:
		return "ISC"
	case Jmp:
		return "jmp"
	case Jsr:
		return "jsr"
	case KIL:
		return "KIL"
	case LAS:
		return "LAS"
	case LAX:
		return "LAX"
	case Lda:
		return "lda"
	case Ldx:
		return "ldx"
	case Ldy:
		return "ldy"
	case Lsr:
		return "lsr"
	case NOP:
		return "NOP"
	case Ora:
		return "ora"
	case Pha:
		return "pha"
	case Php:
		return "php"
	case Pla:
		return "pla"
	case Plp:
		return "plp"
	case RLA:
		return "RLA"
	case Rol:
		return "rol"
	case Ror:
		return "ror"
	case RRA:
		return "RRA"
	case Rti:
		return "rti"
	case Rts:
		return "rts"
	case SAX:
		return "SAX"
	case Sbc:
		return "sbc"
	case SBC:
		return "SBC"
	case Sec:
		return "sec"
	case Sed:
		return "sed"
	case Sei:
		return "sei"
	case SHX:
		return "SHX"
	case SHY:
		return "SHY"
	case SLO:
		return "SLO"
	case SRE:
		return "SRE"
	case Sta:
		return "sta"
	case Stx:
		return "stx"
	case Sty:
		return "sty"
	case TAS:
		return "TAS"
	case Tax:
		return "tax"
	case Tay:
		return "tay"
	case Tsx:
		return "tsx"
	case Txa:
		return "txa"
	case Txs:
		return "txs"
	case Tya:
		return "tya"
	case XAA:
		return "XAA"
	}

	panic(fmt.Sprintf("unrecognised operator %d", operator))
}

// Definition defines each instruction in the instruction set; one per instruction.
type Definition struct {
	OpCode         uint8
	Operator       Operator
	Bytes          int
	Cycles         Cycles
	AddressingMode AddressingMode
	PageSensitive  bool
	Effect         EffectCategory

	// Whether instruction is "undocumented".
	Undocumented bool
}

// String returns a single instruction definition as a string.
func (defn Definition) String() string {
	return fmt.Sprintf("%02x %s +%dbytes (%s cycles) [mode=%d pagesens=%t effect=%d]", defn.OpCode, defn.Operator, defn.Bytes, defn.Cycles.Formatted, defn.AddressingMode, defn.PageSensitive, defn.Effect)
}

// IsBranch returns true if instruction is a branch instruction.
func (defn Definition) IsBranch() bool {
	return defn.AddressingMode == Relative && defn.Effect == Flow
}
