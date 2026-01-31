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

// Operator defines which operation is performed by the opcode. Many opcodes
// can perform the same operation.
type Operator int

// List of valid Operator values.
const (
	NOP Operator = iota
	ADC
	AHX
	ANC
	AND
	ARR
	ASL
	ASR
	AXS
	BCC
	BCS
	BEQ
	BIT
	BMI
	BNE
	BPL
	BRK
	BVC
	BVS
	CLC
	CLD
	CLI
	CLV
	CMP
	CPX
	CPY
	DCP
	DEC
	DEX
	DEY
	EOR
	INC
	INX
	INY
	ISC
	JMP
	JSR
	KIL
	LAS
	LAX
	LDA
	LDX
	LDY
	LSR
	ORA
	PHA
	PHP
	PLA
	PLP
	RLA
	ROL
	ROR
	RRA
	RTI
	RTS
	SAX
	SBC
	SEC
	SED
	SEI
	SHX
	SHY
	SLO
	SRE
	STA
	STX
	STY
	TAS
	TAX
	TAY
	TSX
	TXA
	TXS
	TYA
	XAA
)

func (operator Operator) String() string {
	switch operator {
	case NOP:
		return "nop"
	case ADC:
		return "adc"
	case AHX:
		return "ahx"
	case ANC:
		return "anc"
	case AND:
		return "and"
	case ARR:
		return "arr"
	case ASL:
		return "asl"
	case ASR:
		return "asr"
	case AXS:
		return "axs"
	case BCC:
		return "bcc"
	case BCS:
		return "bcs"
	case BEQ:
		return "beq"
	case BIT:
		return "bit"
	case BMI:
		return "bmi"
	case BNE:
		return "bne"
	case BPL:
		return "bpl"
	case BRK:
		return "brk"
	case BVC:
		return "bvc"
	case BVS:
		return "bvs"
	case CLC:
		return "clc"
	case CLD:
		return "cld"
	case CLI:
		return "cli"
	case CLV:
		return "clv"
	case CMP:
		return "cmp"
	case CPX:
		return "cpx"
	case CPY:
		return "cpy"
	case DCP:
		return "dcp"
	case DEC:
		return "dec"
	case DEX:
		return "dex"
	case DEY:
		return "dey"
	case EOR:
		return "eor"
	case INC:
		return "inc"
	case INX:
		return "inx"
	case INY:
		return "iny"
	case ISC:
		return "isc"
	case JMP:
		return "jmp"
	case JSR:
		return "jsr"
	case KIL:
		return "kil"
	case LAS:
		return "las"
	case LAX:
		return "lax"
	case LDA:
		return "lda"
	case LDX:
		return "ldx"
	case LDY:
		return "ldy"
	case LSR:
		return "lsr"
	case ORA:
		return "ora"
	case PHA:
		return "pha"
	case PHP:
		return "php"
	case PLA:
		return "pla"
	case PLP:
		return "plp"
	case RLA:
		return "rla"
	case ROL:
		return "rol"
	case ROR:
		return "ror"
	case RRA:
		return "rra"
	case RTI:
		return "rti"
	case RTS:
		return "rts"
	case SAX:
		return "sax"
	case SBC:
		return "sbc"
	case SEC:
		return "sec"
	case SED:
		return "sed"
	case SEI:
		return "sei"
	case SHX:
		return "shx"
	case SHY:
		return "shy"
	case SLO:
		return "slo"
	case SRE:
		return "sre"
	case STA:
		return "sta"
	case STX:
		return "stx"
	case STY:
		return "sty"
	case TAS:
		return "tas"
	case TAX:
		return "tax"
	case TAY:
		return "tay"
	case TSX:
		return "tsx"
	case TXA:
		return "txa"
	case TXS:
		return "txs"
	case TYA:
		return "tya"
	case XAA:
		return "xaa"
	default:
		panic(fmt.Sprintf("unrecognised operator %d", operator))
	}
}
