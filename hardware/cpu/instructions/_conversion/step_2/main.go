package main

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
)

type importDefinition struct {
	OpCode         uint8  `json:"opcode"`
	Operator       string `json:"operator"`
	Bytes          int    `json:"bytes"`
	Cycles         int    `json:"cycles"`
	PageSensitive  bool   `json:"pageSensitive"`
	AddressingMode string `json:"addressingMode"`
	Effect         string `json:"effect"`
	Undocumented   bool   `json:"undocumented"`
}

type Definition struct {
	OpCode        uint8
	Operator      string
	Bytes         int
	Cycles        int
	PageSensitive bool

	AddressingMode instructions.AddressingMode
	Effect         instructions.EffectCategory

	Undocumented bool
}

//go:embed "definitions.json"
var definitions_json []byte

func main() {
	var imported []importDefinition
	json.Unmarshal(definitions_json, &imported)

	var definitions []Definition

	for _, imp := range imported {
		def := Definition{
			OpCode:        imp.OpCode,
			Operator:      imp.Operator,
			Bytes:         imp.Bytes,
			Cycles:        imp.Cycles,
			PageSensitive: imp.PageSensitive,
			Undocumented:  imp.Undocumented,
		}

		switch strings.ToLower(imp.AddressingMode) {
		case "implied":
			def.AddressingMode = instructions.Implied
		case "immediate":
			def.AddressingMode = instructions.Immediate
		case "relative":
			def.AddressingMode = instructions.Relative
		case "absolute":
			def.AddressingMode = instructions.Absolute
		case "zeroPage":
			def.AddressingMode = instructions.ZeroPage
		case "indirect":
			def.AddressingMode = instructions.Indirect
		case "indexedindirect":
			def.AddressingMode = instructions.IndexedIndirect
		case "indirectindexed":
			def.AddressingMode = instructions.IndirectIndexed
		case "absoluteindexedx":
			def.AddressingMode = instructions.AbsoluteIndexedX
		case "absoluteindexedy":
			def.AddressingMode = instructions.AbsoluteIndexedY
		case "zeropageindexedx":
			def.AddressingMode = instructions.ZeroPageIndexedX
		case "zeropageindexedy":
			def.AddressingMode = instructions.ZeroPageIndexedY
		}

		switch strings.ToLower(imp.Effect) {
		case "read":
			def.Effect = instructions.Read
		case "write":
			def.Effect = instructions.Write
		case "rmw":
			def.Effect = instructions.RMW
		case "flow":
			def.Effect = instructions.Flow
		case "subroutine":
			def.Effect = instructions.Subroutine
		case "interrupt":
			def.Effect = instructions.Interrupt
		}

		definitions = append(definitions, def)
	}
}
