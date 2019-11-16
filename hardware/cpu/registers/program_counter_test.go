package registers_test

import (
	"gopher2600/hardware/cpu/registers"
	"gopher2600/hardware/cpu/registers/assert"
	"testing"
)

func TestProgramCounter(t *testing.T) {
	// initialisation
	pc := registers.NewProgramCounter(0)
	assert.Assert(t, pc.Address(), 0)

	// loading & addition
	pc.Load(127)
	assert.Assert(t, pc, 127)
	pc.Add(2)
	assert.Assert(t, pc, 129)
}
