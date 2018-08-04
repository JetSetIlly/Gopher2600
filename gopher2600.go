package main

import (
	"flag"
	"fmt"
	"gopher2600/debugger"
	"gopher2600/debugger/colorterm"
	"gopher2600/debugger/ui"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/television"
	"gopher2600/television/sdltv"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
)

const initScript = ".gopher2600/debuggerInit"

func main() {
	mode := flag.String("mode", "DEBUG", "emulation mode: DEBUG, DISASM, RUN, PLAY, FPS, TVFPS")
	termType := flag.String("term", "COLOR", "terminal type to use in debug mode: COLOR, PLAIN")
	flag.Parse()

	cartridgeFile := ""
	if len(flag.Args()) == 1 {
		cartridgeFile = flag.Args()[0]
	} else if len(flag.Args()) > 1 {
		fmt.Println("* too many arguments")
		os.Exit(10)
	}

	switch strings.ToUpper(*mode) {
	case "DEBUG":
		dbg, err := debugger.NewDebugger()
		if err != nil {
			fmt.Printf("* error starting debugger: %s\n", err)
			os.Exit(10)
		}

		// start debugger with choice of interface and cartridge
		var term ui.UserInterface

		switch strings.ToUpper(*termType) {
		case "COLOR":
			term = new(colorterm.ColorTerminal)
		default:
			fmt.Printf("! unknown terminal type (%s) defaulting to plain\n", *termType)
			fallthrough
		case "PLAIN":
			term = nil
		}

		err = dbg.Start(term, cartridgeFile, initScript)
		if err != nil {
			fmt.Printf("* error running debugger: %s\n", err)
			os.Exit(10)
		}
	case "DISASM":
		dsm, err := disassembly.NewDisassembly(cartridgeFile)
		if err != nil {
			switch err.(type) {
			case errors.GopherError:
				// print what disassembly output we do have
				if dsm != nil {
					fmt.Println(dsm.Dump())
				}
			}
			fmt.Printf("* error during disassembly: %s\n", err)
			os.Exit(10)
		}
		fmt.Println(dsm.Dump())
	case "FPS":
		err := fps(cartridgeFile, true)
		if err != nil {
			fmt.Printf("* error starting FPS profiler: %s\n", err)
			os.Exit(10)
		}
	case "TVFPS":
		err := fps(cartridgeFile, false)
		if err != nil {
			fmt.Printf("* error starting TVFPS profiler: %s\n", err)
			os.Exit(10)
		}

	case "PLAY":
		// PLAY is a synonym for RUN
		fallthrough

	case "RUN":
		err := run(cartridgeFile)
		if err != nil {
			fmt.Printf("* error running emulator: %s\n", err)
			os.Exit(10)
		}
	default:
		fmt.Printf("* unknown mode: %s\n", strings.ToUpper(*mode))
		os.Exit(10)
	}
}

func fps(cartridgeFile string, justTheVCS bool) error {
	var tv television.Television
	var err error

	if justTheVCS {
		tv = new(television.DummyTV)
		if tv == nil {
			return fmt.Errorf("error preparing television: %s", err)
		}
	} else {
		tv, err = sdltv.NewSDLTV("NTSC", sdltv.IdealScale)
		if err != nil {
			return fmt.Errorf("error preparing television: %s", err)
		}
	}
	tv.SetVisibility(true)

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return fmt.Errorf("error preparing VCS: %s", err)
	}

	err = vcs.AttachCartridge(cartridgeFile)
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
		stepCycles, _, err := vcs.Step(hardware.NullVideoCycleCallback)
		if err != nil {
			return err
		}
		cycles -= stepCycles
	}

	fmt.Printf("%f fps\n", float64(numOfFrames)/time.Since(startTime).Seconds())

	return nil
}

func run(cartridgeFile string) error {
	tv, err := sdltv.NewSDLTV("NTSC", sdltv.IdealScale)
	if err != nil {
		return fmt.Errorf("error preparing television: %s", err)
	}
	tv.SetVisibility(true)

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return fmt.Errorf("error preparing VCS: %s", err)
	}

	err = vcs.AttachCartridge(cartridgeFile)
	if err != nil {
		return err
	}

	// protecting "running" variable with a mutex
	var runningLock sync.Mutex
	running := true

	err = tv.RegisterCallback(sdltv.ReqOnWindowClose, func() {
		runningLock.Lock()
		running = false
		runningLock.Unlock()
	})
	if err != nil {
		return err
	}

	for {
		runningLock.Lock()
		if !running {
			runningLock.Unlock()
			break
		}
		runningLock.Unlock()

		_, _, err := vcs.Step(hardware.NullVideoCycleCallback)
		if err != nil {
			return err
		}
	}

	return nil
}
