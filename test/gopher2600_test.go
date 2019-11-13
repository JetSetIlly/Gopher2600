package main_test

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/gui"
	"gopher2600/gui/sdldebug"
	"gopher2600/hardware"
	"gopher2600/television"
	"testing"
)

func BenchmarkSDL(b *testing.B) {
	var err error

	tv, err := television.NewTelevision("AUTO")
	if err != nil {
		panic(fmt.Errorf("error preparing television: %s", err))
	}

	scr, err := sdldebug.NewSdlDebug(tv, 1.0)
	if err != nil {
		panic(fmt.Errorf("error preparing screen: %s", err))
	}

	err = scr.SetFeature(gui.ReqSetVisibility, true)
	if err != nil {
		panic(fmt.Errorf("error preparing screen: %s", err))
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		panic(fmt.Errorf("error preparing VCS: %s", err))
	}

	err = vcs.AttachCartridge(cartridgeloader.Loader{Filename: "../roms/ROMs/Pitfall.bin"})
	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for steps := 0; steps < b.N; steps++ {
		err = vcs.Step(nil)
		if err != nil {
			panic(err)
		}
	}
}
