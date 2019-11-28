package registers_test

import (
	"gopher2600/hardware/cpu/registers"
	rtest "gopher2600/hardware/cpu/registers/test"
	"gopher2600/test"
	"testing"
)

func TestProgramCounter(t *testing.T) {
	// initialisation
	pc := registers.NewProgramCounter(0)
	test.Equate(t, pc.Address(), 0)

	// loading & addition
	pc.Load(127)
	rtest.EquateRegisters(t, pc, 127)
	pc.Add(2)
	rtest.EquateRegisters(t, pc, 129)
}
