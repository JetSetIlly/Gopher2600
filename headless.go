package main

import (
	"fmt"
	"headlessVCS/debugger"
	"headlessVCS/hardware"
	"os"
	"runtime/pprof"
	"time"
)

func main() {
	dbg := debugger.NewDebugger()
	err := dbg.Start("flappy.bin")
	if err != nil {
		fmt.Println(err)
		os.Exit(10)
	}
	/*
		err := fps()
		if err != nil {
			fmt.Println(err)
		}
	*/
}

func fps() error {
	vcs := hardware.NewVCS()
	err := vcs.AttachCartridge("flappy.bin")
	if err != nil {
		return err
	}

	const cyclesPerFrame = 19912
	const numOfFrames = 60

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
		result, err := vcs.Step()
		if err != nil {
			fmt.Println(err)
			fmt.Printf("%d cycles completed\n", cycles)
			return nil
		}
		if result.Final {
			cycles -= result.ActualCycles
		}
	}

	fmt.Printf("%f fps\n", float64(numOfFrames)/time.Since(startTime).Seconds())

	return nil
}
