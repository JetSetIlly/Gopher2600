package main

import (
	"flag"
	"fmt"
	"gopher2600/debugger"
	"gopher2600/hardware"
	"gopher2600/television"
	"os"
	"runtime/pprof"
	"strings"
	"time"
)

func main() {
	var mode = flag.String("mode", "DEBUG", "emulation mode: DEBUG, FPS, DISASM")
	flag.Parse()

	switch strings.ToUpper(*mode) {
	case "DEBUG":
		dbg, err := debugger.NewDebugger()
		if err != nil {
			fmt.Printf("* error starting debugger (%s)\n", err)
			os.Exit(10)
		}

		err = dbg.Start("flappy.bin")
		if err != nil {
			fmt.Println(err)
			os.Exit(10)
		}
	case "FPS":
		err := fps()
		if err != nil {
			fmt.Println(err)
			os.Exit(10)
		}
	case "DISASM":
		fmt.Printf("* not yet implemented")
		os.Exit(10)
	default:
		fmt.Printf("* unknown mode (-mode %s)\n", strings.ToUpper(*mode))
		os.Exit(10)
	}

}

func fps() error {
	tv := new(television.DummyTV)
	if tv == nil {
		return fmt.Errorf("error creating television for fps profiler")
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return fmt.Errorf("error starting fps profiler (%s)", err)
	}

	err = vcs.AttachCartridge("flappy.bin")
	if err != nil {
		return err
	}

	const cyclesPerFrame = 19912
	const numOfFrames = 180

	f, err := os.Create("cpu.profile")
	if err != nil {
		return err
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		return err
	}
	defer pprof.StopCPUProfile()

	cycles := cyclesPerFrame * numOfFrames
	startTime := time.Now()
	for cycles > 0 {
		stepCycles, _, err := vcs.Step()
		if err != nil {
			fmt.Println(err)
			fmt.Printf("%d cycles completed\n", cycles)
			return nil
		}
		cycles -= stepCycles
	}

	fmt.Printf("%f fps\n", float64(numOfFrames)/time.Since(startTime).Seconds())

	return nil
}
