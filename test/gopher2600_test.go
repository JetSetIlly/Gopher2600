package main_test

import (
	"fmt"
	"gopher2600/gui"
	"gopher2600/gui/sdl"
	"gopher2600/hardware"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"testing"
)

func BenchmarkSDL(b *testing.B) {
	var err error

	tv, err := sdl.NewGUI("NTSC", 1.0, nil)
	if err != nil {
		panic(fmt.Errorf("error preparing television: %s", err))
	}

	err = tv.SetFeature(gui.ReqSetVisibility, true)
	if err != nil {
		panic(fmt.Errorf("error preparing television: %s", err))
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		panic(fmt.Errorf("error preparing VCS: %s", err))
	}

	err = vcs.AttachCartridge(memory.CartridgeLoader{Filename: "../roms/ROMs/Pitfall.bin"})
	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for steps := 0; steps < b.N; steps++ {
		_, err = vcs.Step(func(*result.Instruction) error { return nil })
		if err != nil {
			panic(err)
		}
	}
}
