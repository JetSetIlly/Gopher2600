package memory

import (
	"fmt"
	"gopher2600/hardware/memory/memorymap"
	"strings"
)

// PIA defines the information for and operation allowed for PIA PIA
type PIA struct {
	DebuggerBus
	CPUBus

	origin uint16
	memtop uint16
	memory []uint8
}

// newPIA is the preferred method of initialisation for the PIA pia memory area
func newPIA() *PIA {
	pia := &PIA{
		origin: memorymap.OriginPIA,
		memtop: memorymap.MemtopPIA,
	}

	// allocate the mininmal amount of memory
	pia.memory = make([]uint8, pia.memtop-pia.origin+1)

	return pia
}

func (pia PIA) String() string {
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
func (pia PIA) Peek(address uint16) (uint8, error) {
	oa := address - pia.origin
	return pia.memory[oa], nil
}

// Poke is the implementation of memory.DebuggerBus
func (pia PIA) Poke(address uint16, value uint8) error {
	oa := address - pia.origin
	pia.memory[oa] = value
	return nil
}

// Read is an implementatio of memory.ChipBus
func (pia PIA) Read(address uint16) (uint8, error) {
	return pia.memory[pia.origin|address^pia.origin], nil
}

// Write is an implementatio of memory.ChipBus
func (pia *PIA) Write(address uint16, data uint8) error {
	pia.memory[pia.origin|address^pia.origin] = data
	return nil
}
