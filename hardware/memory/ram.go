package memory

import (
	"fmt"
	"gopher2600/hardware/memory/bus"
	"gopher2600/hardware/memory/memorymap"
	"strings"
)

// RAM represents the 128bytes of RAM in the PIA 6532 chip, found in the Atari
// VCS.
type RAM struct {
	bus.DebuggerBus
	bus.CPUBus
	memory []uint8
}

// newRAM is the preferred method of initialisation for the RAM memory area
func newRAM() *RAM {
	ram := &RAM{}

	// allocate the mininmal amount of memory
	ram.memory = make([]uint8, memorymap.MemtopRAM-memorymap.OriginRAM+1)

	return ram
}

func (ram RAM) String() string {
	s := strings.Builder{}
	s.WriteString("      -0 -1 -2 -3 -4 -5 -6 -7 -8 -9 -A -B -C -D -E -F\n")
	s.WriteString("    ---- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --\n")
	for y := 0; y < 8; y++ {
		s.WriteString(fmt.Sprintf("%X- | ", y+8))
		for x := 0; x < 16; x++ {
			s.WriteString(fmt.Sprintf(" %02x", ram.memory[uint16((y*16)+x)]))
		}
		s.WriteString("\n")
	}
	return strings.Trim(s.String(), "\n")
}

// Peek is the implementation of memory.DebuggerBus
func (ram RAM) Peek(address uint16) (uint8, error) {
	return ram.Read(address)
}

// Poke is the implementation of memory.DebuggerBus
func (ram RAM) Poke(address uint16, value uint8) error {
	return ram.Write(address, value)
}

// Read is an implementatio of memory.ChipBus
func (ram RAM) Read(address uint16) (uint8, error) {
	return ram.memory[address^memorymap.OriginRAM], nil
}

// Write is an implementatio of memory.ChipBus
func (ram *RAM) Write(address uint16, data uint8) error {
	ram.memory[address^memorymap.OriginRAM] = data
	return nil
}
