package main

import (
	"flag"
	"fmt"
	"gopher2600/debugger"
	"gopher2600/debugger/colorterm"
	"gopher2600/debugger/console"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/gui/sdl"
	"gopher2600/hardware"
	"gopher2600/playmode"
	"gopher2600/regression"
	"gopher2600/television"
	"io"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"time"
)

const defaultInitScript = ".gopher2600/debuggerInit"

func main() {
	progName := path.Base(os.Args[0])

	progFlags := flag.NewFlagSet(progName, flag.ExitOnError)
	progFlags.Parse(os.Args[1:])

	if len(progFlags.Args()) == 0 {
		fmt.Println("* mode or cartridge required")
		os.Exit(2)
	}

	mode := strings.ToUpper(progFlags.Arg(0))
	modeArgPos := 1
	modeFlags := flag.NewFlagSet(fmt.Sprintf("%s %s", progName, mode), flag.ExitOnError)
	modeFlagsParse := func() {
		if len(progFlags.Args()) >= modeArgPos {
			modeFlags.Parse(progFlags.Args()[modeArgPos:])
		}
	}

	switch mode {
	default:
		// RUN is the default mode
		modeArgPos = 0
		fallthrough

	case "RUN":
		fallthrough

	case "PLAY":
		tvMode := modeFlags.String("tv", "NTSC", "television specification: NTSC, PAL")
		scaling := modeFlags.Float64("scale", 3.0, "television scaling")
		stable := modeFlags.Bool("stable", true, "wait for stable frame before opening display")
		modeFlagsParse()

		switch len(modeFlags.Args()) {
		case 0:
			fmt.Println("* 2600 cartridge required")
			os.Exit(2)
		case 1:
			err := playmode.Play(modeFlags.Arg(0), *tvMode, float32(*scaling), *stable)
			if err != nil {
				fmt.Printf("* error running emulator: %s\n", err)
				os.Exit(2)
			}
		default:
			fmt.Printf("* too many arguments for %s mode\n", mode)
			os.Exit(2)
		}

	case "DEBUG":
		termType := modeFlags.String("term", "COLOR", "terminal type to use in debug mode: COLOR, PLAIN")
		initScript := modeFlags.String("initscript", defaultInitScript, "terminal type to use in debug mode: COLOR, PLAIN")
		modeFlagsParse()

		tv, err := sdl.NewGUI("NTSC", 2.0)
		if err != nil {
			fmt.Printf("* error preparing television: %s", err)
			os.Exit(2)
		}

		dbg, err := debugger.NewDebugger(tv)
		if err != nil {
			fmt.Printf("* error starting debugger: %s\n", err)
			os.Exit(2)
		}

		// start debugger with choice of interface and cartridge
		var term console.UserInterface

		switch strings.ToUpper(*termType) {
		default:
			fmt.Printf("! unknown terminal type (%s) defaulting to plain\n", *termType)
			fallthrough
		case "PLAIN":
			term = nil
		case "COLOR":
			term = new(colorterm.ColorTerminal)
		}

		switch len(modeFlags.Args()) {
		case 0:
			// it's okay if DEBUG mode is started with no cartridges
			fallthrough
		case 1:
			err := dbg.Start(term, modeFlags.Arg(0), *initScript)
			if err != nil {
				fmt.Printf("* error running debugger: %s\n", err)
				os.Exit(2)
			}
		default:
			fmt.Printf("* too many arguments for %s mode\n", mode)
			os.Exit(2)
		}

	case "DISASM":
		modeFlagsParse()

		switch len(modeFlags.Args()) {
		case 0:
			fmt.Println("* 2600 cartridge required")
			os.Exit(2)
		case 1:
			dsm, err := disassembly.NewDisassembly(modeFlags.Arg(0))
			if err != nil {
				switch err.(type) {
				case errors.FormattedError:
					// print what disassembly output we do have
					if dsm != nil {
						dsm.Dump(os.Stdout)
					}
				}
				fmt.Printf("* error during disassembly: %s\n", err)
				os.Exit(2)
			}
			dsm.Dump(os.Stdout)
		default:
			fmt.Printf("* too many arguments for %s mode\n", mode)
			os.Exit(2)
		}

	case "FPS":
		display := modeFlags.Bool("display", false, "display TV output: boolean")
		tvMode := modeFlags.String("tv", "NTSC", "television specification: NTSC, PAL")
		scaling := modeFlags.Float64("scale", 3.0, "television scaling")
		runTime := modeFlags.String("time", "5s", "run duration (note: there is a 2s overhead)")
		profile := modeFlags.Bool("profile", false, "perform cpu and memory profiling")
		modeFlagsParse()

		switch len(modeFlags.Args()) {
		case 0:
			fmt.Println("* 2600 cartridge required")
			os.Exit(2)
		case 1:
			err := fps(*profile, modeFlags.Arg(0), *display, *tvMode, float32(*scaling), *runTime)
			if err != nil {
				fmt.Printf("* error starting fps profiler: %s\n", err)
				os.Exit(2)
			}
		default:
			fmt.Printf("* too many arguments for %s mode\n", mode)
			os.Exit(2)
		}

	case "REGRESS":
		subMode := strings.ToUpper(progFlags.Arg(1))
		modeArgPos++
		switch subMode {
		default:
			modeArgPos-- // undo modeArgPos adjustment
			fallthrough
		case "RUN":
			verbose := modeFlags.Bool("verbose", false, "display details of each test")
			failOnError := modeFlags.Bool("fail", false, "fail on error: boolean")
			modeFlagsParse()

			var output io.Writer
			if *verbose == true {
				output = os.Stdout
			}

			switch len(modeFlags.Args()) {
			case 0:
				succeed, fail, err := regression.RegressRunTests(output, *failOnError)
				if err != nil {
					fmt.Printf("* error during regression tests: %s\n", err)
					os.Exit(2)
				}
				fmt.Printf("regression tests: %d succeed, %d fail\n", succeed, fail)
			default:
				fmt.Printf("* too many arguments for %s mode\n", mode)
				os.Exit(2)
			}

		case "DELETE":
			modeFlagsParse()

			switch len(modeFlags.Args()) {
			case 0:
				fmt.Println("* 2600 cartridge required")
				os.Exit(2)
			case 1:
				err := regression.RegressDeleteCartridge(modeFlags.Arg(0))
				if err != nil {
					fmt.Printf("* error deleting regression entry: %s\n", err)
					os.Exit(2)
				}
				fmt.Printf("! deleted %s from regression database\n", path.Base(modeFlags.Arg(0)))
			default:
				fmt.Printf("* too many arguments for %s mode\n", mode)
				os.Exit(2)
			}

		case "ADD":
			tvMode := modeFlags.String("tv", "NTSC", "television specification: NTSC, PAL")
			numFrames := modeFlags.Int("frames", 10, "number of frames to run")
			modeFlagsParse()

			switch len(modeFlags.Args()) {
			case 0:
				fmt.Println("* 2600 cartridge required")
				os.Exit(2)
			case 1:
				err := regression.RegressAddCartridge(modeFlags.Arg(0), *tvMode, *numFrames)
				if err != nil {
					fmt.Printf("* error adding regression test: %s\n", err)
					os.Exit(2)
				}
				fmt.Printf("! added %s to regression database\n", path.Base(modeFlags.Arg(0)))
			default:
				fmt.Printf("* too many arguments for %s mode\n", mode)
				os.Exit(2)
			}
		case "UPDATE":
			tvMode := modeFlags.String("tv", "NTSC", "television specification: NTSC, PAL")
			numFrames := modeFlags.Int("frames", 10, "number of frames to run")
			modeFlagsParse()

			switch len(modeFlags.Args()) {
			case 0:
				fmt.Println("* 2600 cartridge required")
				os.Exit(2)
			case 1:
				err := regression.RegressUpdateCartridge(modeFlags.Arg(0), *tvMode, *numFrames)
				if err != nil {
					fmt.Printf("* error updating regression test: %s\n", err)
					os.Exit(2)
				}
				fmt.Printf("! updated %s in regression database\n", path.Base(modeFlags.Arg(0)))
			default:
				fmt.Printf("* too many arguments for %s mode\n", mode)
				os.Exit(2)
			}
		}
	}
}

func fps(profile bool, cartridgeFile string, display bool, tvMode string, scaling float32, runTime string) error {
	var tv television.Television
	var err error

	if display {
		tv, err = sdl.NewGUI(tvMode, scaling)
		if err != nil {
			return fmt.Errorf("error preparing television: %s", err)
		}

		err = tv.(gui.GUI).SetFeature(gui.ReqSetVisibility, true)
		if err != nil {
			return fmt.Errorf("error preparing television: %s", err)
		}
	} else {
		tv, err = television.NewHeadlessTV("NTSC")
		if err != nil {
			return fmt.Errorf("error preparing television: %s", err)
		}
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return fmt.Errorf("error preparing VCS: %s", err)
	}

	err = vcs.AttachCartridge(cartridgeFile)
	if err != nil {
		return err
	}

	// write cpu profile
	if profile {
		f, err := os.Create("cpu.profile")
		if err != nil {
			return err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}

	// get starting frame number
	tvState, err := tv.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}
	startFrame := tvState.(int)

	// run for specified period of time

	// -- parse supplied duration
	duration, err := time.ParseDuration(runTime)
	if err != nil {
		return err
	}

	// -- setup trigger that expires when duration has elapsed
	var running atomic.Value
	running.Store(0)

	go func() {
		// force a two second leadtime to allow framerate to settle down
		time.AfterFunc(2*time.Second, func() {
			tvState, err = tv.GetState(television.ReqFramenum)
			if err != nil {
				panic(err)
			}

			startFrame = tvState.(int)

			time.AfterFunc(duration, func() {
				running.Store(-1)
			})
		})
	}()

	// -- run until specified time elapses (running is changed to -1)
	err = vcs.Run(&running)
	if err != nil {
		return err
	}

	// get ending frame number
	tvState, err = vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}
	endFrame := tvState.(int)

	// calculate and display frames-per-second
	frameCount := endFrame - startFrame
	fps := float64(frameCount) / duration.Seconds()
	fmt.Printf("%.2f fps (%d frames in %.2f seconds)\n", fps, frameCount, duration.Seconds())

	// write memory profile
	if profile {
		f, err := os.Create("mem.profile")
		if err != nil {
			return err
		}
		runtime.GC()
		err = pprof.WriteHeapProfile(f)
		if err != nil {
			return fmt.Errorf("could not write memory profile: %s", err)
		}
		f.Close()
	}

	return nil
}
