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

	origin uint16
	memtop uint16
	memory []uint8
}

// newRAM is the preferred method of initialisation for the RAM memory area
func newRAM() *RAM {
	pia := &RAM{
		origin: memorymap.OriginRAM,
		memtop: memorymap.MemtopRAM,
	}

	// allocate the mininmal amount of memory
	pia.memory = make([]uint8, pia.memtop-pia.origin+1)

	return pia
}

func (pia RAM) String() string {
	s := strings.Builder{}
	s.WriteString("      -0 -1 -2 -3 -4 -5 -6 -7 -8 -9 -A -B -C -D -E -F\n")
	s.WriteString("    ---- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --\n")
	for y := 0; y < 8; y++ {
		s.WriteString(fmt.Sprintf("%X- | ", y+8))
		for x := 0; x < 16; x++ {
			s.WriteString(fmt.Sprintf(" %02x", pia.memory[uint16((y*16)+x)]))
		}
		s.WriteString("\n")
	}
	return strings.Trim(s.String(), "\n")
}

// Peek is the implementation of memory.DebuggerBus
func (pia RAM) Peek(address uint16) (uint8, error) {
	oa := address - pia.origin
	return pia.memory[oa], nil
}

// Poke is the implementation of memory.DebuggerBus
func (pia RAM) Poke(address uint16, value uint8) error {
	oa := address - pia.origin
	pia.memory[oa] = value
	return nil
}

// Read is an implementatio of memory.ChipBus
func (pia RAM) Read(address uint16) (uint8, error) {
	return pia.memory[pia.origin|address^pia.origin], nil
}

// Write is an implementatio of memory.ChipBus
func (pia *RAM) Write(address uint16, data uint8) error {
	pia.memory[pia.origin|address^pia.origin] = data
	return nil
}
