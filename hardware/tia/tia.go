package tia

import (
	"fmt"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia/polycounter"
)

// TIA contains all the sub-components of the VCS video chip
type TIA struct {
	mem   *memory.VCSMemory
	hsync *polycounter.Polycounter
}

func (tia TIA) String() string {
	return fmt.Sprintf("HSYNC: %s\n", tia.hsync)
}

// NewTIA is the preferred method of initialisation for the TIA structure
func NewTIA(mem *memory.VCSMemory) *TIA {
	tia := new(TIA)
	if tia == nil {
		return nil
	}
	tia.mem = mem
	tia.hsync = polycounter.New6BitPolycounter("010100")
	if tia.hsync == nil {
		return nil
	}
	return tia
}

// StepVideoCycle moves the video state forward one cycle
func (tia *TIA) StepVideoCycle() {
	tia.hsync.Tick()
}
