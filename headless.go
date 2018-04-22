package main

import (
	"fmt"
	"headlessVCS/hardware"
	"log"
	"os"
	"runtime/pprof"
	"time"
)

func main() {
	dbg := NewDebugger()

	err := dbg.vcs.AttachCartridge("flappy.bin")
	if err != nil {
		fmt.Println(err)
		os.Exit(10)
	}

	dbg.fps()
	dbg.vcs.Reset()

	/*
		err = dbg.inputLoop()
		if err != nil {
			fmt.Println(err)
			os.Exit(10)
		}
	*/
}

// Debugger is the basic debugging frontend for the emulation
type Debugger struct {
	vcs       *hardware.VCS
	running   bool
	inputBuff []byte
}

// NewDebugger is the preferred method of initialisation for the Debugger structure
func NewDebugger() *Debugger {
	dbg := new(Debugger)
	dbg.vcs = hardware.NewVCS()
	dbg.inputBuff = make([]byte, 255)
	return dbg
}

func (dbg *Debugger) fps() {
	const cyclesPerFrame = 19912
	const numOfFrames = 50

	f, err := os.Create("cpu.profile")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	cycles := cyclesPerFrame * numOfFrames
	startTime := time.Now()
	for cycles > 0 {
		result, err := dbg.vcs.Step()
		if err != nil {
			fmt.Println(err)
			fmt.Printf("%d cycles completed\n", cycles)
			return
		}
		if result.Final {
			cycles -= result.ActualCycles
		}
	}

	fmt.Printf("%f fps\n", float64(numOfFrames)/time.Since(startTime).Seconds())
}

func (dbg *Debugger) inputLoop() error {
	breakpoint := true

	dbg.running = true
	for dbg.running == true {
		if breakpoint {
			fmt.Printf("> ")
			_, err := os.Stdin.Read(dbg.inputBuff)
			if err != nil {
				return err
			}
			breakpoint = false

			// TODO: parse user input
		}

		result, err := dbg.vcs.Step()
		if err != nil {
			return err
		}
		fmt.Println(result)

		// TODO: check for breakpoints
		breakpoint = true
	}

	return nil
}
