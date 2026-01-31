package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
)

type Instruction struct {
	OpCode         uint8  `json:"opcode"`
	Operator       string `json:"operator"`
	Bytes          int    `json:"bytes"`
	Cycles         int    `json:"cycles"`
	AddressingMode string `json:"addressingMode"`
	PageSensitive  bool   `json:"pageSensitive"`
	Effect         string `json:"effect"`
	Undocumented   bool   `json:"undocumented"`
}

func main() {
	var newDefs []Instruction
	for _, def := range instructions.Definitions {
		newDef := Instruction{
			OpCode:         def.OpCode,
			Operator:       strings.ToLower(def.Operator.String()),
			Bytes:          def.Bytes,
			Cycles:         def.Cycles.Value,
			AddressingMode: strings.ToLower(def.AddressingMode.String()),
			PageSensitive:  def.PageSensitive,
			Effect:         strings.ToLower(def.Effect.String()),
			Undocumented:   def.Undocumented,
		}
		newDefs = append(newDefs, newDef)
	}

	b, err := json.MarshalIndent(newDefs, "", "  ")
	if err != nil {
		panic(err)
	}

	f, err := os.Create("definitions.json")
	if err != nil {
		panic(err)
	}
	f.Write(b)
	err = f.Close()
	if err != nil {
		panic(err)
	}
}
