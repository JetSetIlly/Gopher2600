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

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Definition defines each instruction in the instruction set; one per instruction
type Definition struct {
	OpCode         uint8
	Operator       Operator
	Bytes          int
	Cycles         int
	AddressingMode AddressingMode
	PageSensitive  bool
	Effect         Category
	Undocumented   bool
	Stability      Stability
}

// String returns a single instruction definition as a string
func (defn Definition) String() string {
	return fmt.Sprintf("%02x %s +%dbytes (%d cycles) [mode=%s pagesens=%t effect=%s]", defn.OpCode, defn.Operator, defn.Bytes, defn.Cycles, defn.AddressingMode, defn.PageSensitive, defn.Effect)
}

// IsBranch returns true if instruction is a branch instruction
func (defn Definition) IsBranch() bool {
	return defn.AddressingMode == Relative && defn.Effect == Flow
}

type importDefinition struct {
	OpCode         uint8  `json:"opcode"`
	Operator       string `json:"operator"`
	Bytes          int    `json:"bytes"`
	Cycles         int    `json:"cycles"`
	PageSensitive  bool   `json:"pageSensitive"`
	AddressingMode string `json:"addressingMode"`
	Category       string `json:"category"`
	Undocumented   bool   `json:"undocumented"`
	Stability      string `json:"stability,omitempty"`
}

//go:embed "definitions.json"
var definitions_json []byte

// definitions of all the 256 instructions
var Definitions []Definition

func init() {
	var imported []importDefinition
	err := json.Unmarshal(definitions_json, &imported)
	if err != nil {
		panic(fmt.Sprintf("CPU instruction defintions: %s", err.Error()))
	}

	if len(imported) != 256 {
		panic("CPU instruction definitions is incomplete")
	}

	for _, imp := range imported {
		def := Definition{
			OpCode:        imp.OpCode,
			Bytes:         imp.Bytes,
			Cycles:        imp.Cycles,
			PageSensitive: imp.PageSensitive,
			Undocumented:  imp.Undocumented,
		}

		switch strings.ToLower(imp.Operator) {
		case "nop":
			def.Operator = NOP
		case "adc":
			def.Operator = ADC
		case "sha":
			def.Operator = SHA
		case "anc":
			def.Operator = ANC
		case "and":
			def.Operator = AND
		case "arr":
			def.Operator = ARR
		case "asl":
			def.Operator = ASL
		case "alr":
			def.Operator = ALR
		case "bcc":
			def.Operator = BCC
		case "bcs":
			def.Operator = BCS
		case "beq":
			def.Operator = BEQ
		case "bit":
			def.Operator = BIT
		case "bmi":
			def.Operator = BMI
		case "bne":
			def.Operator = BNE
		case "bpl":
			def.Operator = BPL
		case "brk":
			def.Operator = BRK
		case "bvc":
			def.Operator = BVC
		case "bvs":
			def.Operator = BVS
		case "clc":
			def.Operator = CLC
		case "cld":
			def.Operator = CLD
		case "cli":
			def.Operator = CLI
		case "clv":
			def.Operator = CLV
		case "cmp":
			def.Operator = CMP
		case "cpx":
			def.Operator = CPX
		case "cpy":
			def.Operator = CPY
		case "dcp":
			def.Operator = DCP
		case "dec":
			def.Operator = DEC
		case "dex":
			def.Operator = DEX
		case "dey":
			def.Operator = DEY
		case "eor":
			def.Operator = EOR
		case "inc":
			def.Operator = INC
		case "inx":
			def.Operator = INX
		case "iny":
			def.Operator = INY
		case "isc":
			def.Operator = ISC
		case "jmp":
			def.Operator = JMP
		case "jsr":
			def.Operator = JSR
		case "jam":
			def.Operator = JAM
		case "las":
			def.Operator = LAS
		case "lax":
			def.Operator = LAX
		case "lda":
			def.Operator = LDA
		case "ldx":
			def.Operator = LDX
		case "ldy":
			def.Operator = LDY
		case "lsr":
			def.Operator = LSR
		case "ora":
			def.Operator = ORA
		case "pha":
			def.Operator = PHA
		case "php":
			def.Operator = PHP
		case "pla":
			def.Operator = PLA
		case "plp":
			def.Operator = PLP
		case "rla":
			def.Operator = RLA
		case "rol":
			def.Operator = ROL
		case "ror":
			def.Operator = ROR
		case "rra":
			def.Operator = RRA
		case "rti":
			def.Operator = RTI
		case "rts":
			def.Operator = RTS
		case "sax":
			def.Operator = SAX
		case "sbx":
			def.Operator = SBX
		case "sbc":
			def.Operator = SBC
		case "sec":
			def.Operator = SEC
		case "sed":
			def.Operator = SED
		case "sei":
			def.Operator = SEI
		case "shx":
			def.Operator = SHX
		case "shy":
			def.Operator = SHY
		case "slo":
			def.Operator = SLO
		case "sre":
			def.Operator = SRE
		case "sta":
			def.Operator = STA
		case "stx":
			def.Operator = STX
		case "sty":
			def.Operator = STY
		case "tas":
			def.Operator = TAS
		case "tax":
			def.Operator = TAX
		case "tay":
			def.Operator = TAY
		case "tsx":
			def.Operator = TSX
		case "txa":
			def.Operator = TXA
		case "txs":
			def.Operator = TXS
		case "tya":
			def.Operator = TYA
		case "ane":
			def.Operator = ANE
		default:
			panic(fmt.Sprintf("unknown operator: %s", imp.Operator))
		}

		switch strings.ToLower(imp.AddressingMode) {
		case "implied":
			def.AddressingMode = Implied
		case "immediate":
			def.AddressingMode = Immediate
		case "relative":
			def.AddressingMode = Relative
		case "absolute":
			def.AddressingMode = Absolute
		case "indirect":
			def.AddressingMode = Indirect
		case "preindexed":
			def.AddressingMode = PreIndexed
		case "postindexed":
			def.AddressingMode = PostIndexed
		case "absolutex":
			def.AddressingMode = AbsoluteX
		case "absolutey":
			def.AddressingMode = AbsoluteY
		default:
			panic(fmt.Sprintf("unknown addressing mode: %s", imp.AddressingMode))
		}

		switch strings.ToLower(imp.Category) {
		case "read":
			def.Effect = Read
		case "write":
			def.Effect = Write
		case "modify":
			def.Effect = Modify
		case "flow":
			def.Effect = Flow
		case "subroutine":
			def.Effect = Subroutine
		case "interrupt":
			def.Effect = Interrupt
		default:
			panic(fmt.Sprintf("unknown cateogry: %s", imp.Category))
		}

		switch strings.ToLower(imp.Stability) {
		case "stable", "":
			def.Stability = Stable
		case "unstable":
			def.Stability = Unstable
		case "magic":
			def.Stability = Magic
		default:
			panic(fmt.Sprintf("unknown stability assessment: %s", imp.Stability))
		}

		Definitions = append(Definitions, def)
	}

	sort.Slice(Definitions, func(i, j int) bool {
		return Definitions[i].OpCode < Definitions[j].OpCode
	})
}
