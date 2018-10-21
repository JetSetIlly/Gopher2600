package main_test

import (
	"fmt"
	"gopher2600/hardware"
	"gopher2600/television"
	"testing"
)

func BenchmarkCPU(b *testing.B) {
	var err error

	tv := new(television.DummyTV)
	if tv == nil {
		panic(fmt.Errorf("error preparing television: %s", err))
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		panic(fmt.Errorf("error preparing VCS: %s", err))
	}

	err = vcs.AttachCartridge("roms/ROMs/Pitfall.bin")
	if err != nil {
		panic(err)
	}

	for steps := 1000000; steps >= 0; steps-- {
		_, _, err = vcs.Step(hardware.StubVideoCycleCallback)
		if err != nil {
			panic(err)
		}
	}
}
