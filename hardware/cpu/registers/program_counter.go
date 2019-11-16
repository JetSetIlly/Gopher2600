package registers

import "fmt"

// ProgramCounter represents the PC register in the 6502/6507 CPU
type ProgramCounter struct {
	value uint16
}

// NewProgramCounter is the preferred method of initialisation for ProgramCounter
func NewProgramCounter(val uint16) *ProgramCounter {
	return &ProgramCounter{value: val}
}

// Label returns an identifying string for the PC
func (pc ProgramCounter) Label() string {
	return "PC"
}

func (pc ProgramCounter) String() string {
	return fmt.Sprintf("%#04x", pc.value)
}

// FormatValue formats an arbitary value to look like a PC value
func (pc ProgramCounter) FormatValue(val interface{}) string {
	return fmt.Sprintf("%#04x", val)
}

// CurrentValue returns the current value of the PC as an integer (wrapped as a generic value)
func (pc ProgramCounter) CurrentValue() interface{} {
	return int(pc.value)
}

// Address returns the current value of the PC as a a value of type uint16
func (pc *ProgramCounter) Address() uint16 {
	return pc.value
}

// Load a value into the PC
func (pc *ProgramCounter) Load(val uint16) {
	pc.value = val
}

// Add a value to the PC
func (pc *ProgramCounter) Add(val uint16) (carry, overflow bool) {
	v := pc.value
	pc.value += val
	return pc.value < v, false
}
