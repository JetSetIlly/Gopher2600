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

type nop struct{}

func (*nop) Write(p []byte) (n int, err error) {
	return 0, nil
}

func main() {
	progName := path.Base(os.Args[0])

	var mode string
	var modeArgPos int
	var modeFlags *flag.FlagSet
	var modeFlagsParse func()

	progModes := []string{"RUN", "PLAY", "DEBUG", "DISASM", "FPS", "REGRESS"}
	defaultMode := "RUN"

	progFlags := flag.NewFlagSet(progName, flag.ContinueOnError)

	// prevent Parse() from outputting it's own error messages
	progFlags.SetOutput(&nop{})

	err := progFlags.Parse(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			fmt.Printf("available modes: %s\n", strings.Join(progModes, ", "))
			fmt.Printf("default: %s\n", defaultMode)
			os.Exit(2)
		}

		// flags have been set that are not recognised. default to the RUN mode
		// and try again
		mode = defaultMode
		modeArgPos = 0
		modeFlags = flag.NewFlagSet(fmt.Sprintf("%s %s", progName, mode), flag.ExitOnError)
		modeFlagsParse = func() {
			if len(progFlags.Args()) >= modeArgPos {
				modeFlags.Parse(os.Args[1:])
			}
		}
	} else {
		switch progFlags.NArg() {
		case 0:
			// no arguments at all. suggest that a cartridge is required
			fmt.Println("* 2600 cartridge required")
			os.Exit(2)
		case 1:
			// a single argument has been supplied. assume it's a cartridge
			// name and set the mode to the default mode ...
			mode = defaultMode
			modeArgPos = 0

			// ... unless it apears in the list of modes. in which case, the
			// single argument is a specified mode. let the mode switch below
			// handle what to do next.
			arg := strings.ToUpper(progFlags.Arg(0))
			for i := range progModes {
				if progModes[i] == arg {
					mode = arg
					modeArgPos = 1
					break
				}
			}
		default:
			// many arguments have been supplied. the first argument must be
			// the mode (the switch below will compalin if it's invalid)
			mode = strings.ToUpper(progFlags.Arg(0))
			modeArgPos = 1
		}

		// all modes can have their own sets of flags. the following prepares
		// the foundations.
		modeFlags = flag.NewFlagSet(fmt.Sprintf("%s %s", progName, mode), flag.ExitOnError)
		modeFlagsParse = func() {
			if len(progFlags.Args()) >= modeArgPos {
				modeFlags.Parse(progFlags.Args()[modeArgPos:])
			}
		}
	}

	switch mode {
	default:
		fmt.Printf("* %s mode unrecognised\n", mode)
		os.Exit(2)

	case "RUN":
		fallthrough

	case "PLAY":
		tvType := modeFlags.String("tv", "NTSC", "television specification: NTSC, PAL")
		scaling := modeFlags.Float64("scale", 3.0, "television scaling")
		stable := modeFlags.Bool("stable", true, "wait for stable frame before opening display")
		record := modeFlags.Bool("record", false, "record user input to a file")
		recording := modeFlags.String("recording", "", "the file to use for recording/playback")
		modeFlagsParse()

		switch len(modeFlags.Args()) {
		case 0:
			if *recording == "" {
				fmt.Println("* 2600 cartridge required")
				os.Exit(2)
			}
			fallthrough
		case 1:
			err := playmode.Play(modeFlags.Arg(0), *tvType, float32(*scaling), *stable, *recording, *record)
			if err != nil {
				fmt.Println(err)
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

		dbg, err := debugger.NewDebugger()
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
			err := dbg.Start(term, *initScript, modeFlags.Arg(0))
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
			dsm, err := disassembly.FromCartrige(modeFlags.Arg(0))
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
		tvType := modeFlags.String("tv", "NTSC", "television specification: NTSC, PAL")
		scaling := modeFlags.Float64("scale", 3.0, "television scaling")
		runTime := modeFlags.String("time", "5s", "run duration (note: there is a 2s overhead)")
		profile := modeFlags.Bool("profile", false, "perform cpu and memory profiling")
		modeFlagsParse()

		switch len(modeFlags.Args()) {
		case 0:
			fmt.Println("* 2600 cartridge required")
			os.Exit(2)
		case 1:
			err := fps(*profile, modeFlags.Arg(0), *display, *tvType, float32(*scaling), *runTime)
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
			modeFlagsParse()

			var output io.Writer
			if *verbose == true {
				output = os.Stdout
			}

			switch len(modeFlags.Args()) {
			case 0:
				succeed, fail, err := regression.RegressRunTests(output)
				if err != nil {
					fmt.Printf("* error during regression tests: %s\n", err)
					os.Exit(2)
				}
				fmt.Printf("regression tests: %d succeed, %d fail\n", succeed, fail)
			default:
				fmt.Printf("* too many arguments for %s mode\n", mode)
				os.Exit(2)
			}

		case "LIST":
			var output io.Writer
			output = os.Stdout
			err := regression.RegressList(output)
			if err != nil {
				fmt.Printf("* error during regression listing: %s\n", err)
				os.Exit(2)
			}

		case "DELETE":
			modeFlagsParse()

			switch len(modeFlags.Args()) {
			case 0:
				fmt.Println("* 2600 cartridge required")
				os.Exit(2)
			case 1:
				err := regression.RegressDelete(modeFlags.Arg(0))
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
			tvType := modeFlags.String("tv", "NTSC", "television specification: NTSC, PAL")
			numFrames := modeFlags.Int("frames", 10, "number of frames to run")
			modeFlagsParse()

			switch len(modeFlags.Args()) {
			case 0:
				fmt.Println("* 2600 cartridge or playback file required")
				os.Exit(2)
			case 1:
				// TODO: adding different record types
				newRecord := &regression.FrameRecord{
					CartridgeFile: modeFlags.Arg(0),
					TVtype:        *tvType,
					NumFrames:     *numFrames}

				err := regression.RegressAdd(newRecord)
				if err != nil {
					fmt.Printf("* error adding regression test: %s\n", err)
					os.Exit(2)
				}
				fmt.Printf("! added %s to regression database\n", path.Base(modeFlags.Arg(0)))
			default:
				fmt.Printf("* too many arguments for %s mode\n", mode)
				os.Exit(2)
			}
		}
	}
}

func fps(profile bool, cartridgeFile string, display bool, tvType string, scaling float32, runTime string) error {
	var fpstv television.Television
	var err error

	if display {
		fpstv, err = sdl.NewGUI(tvType, scaling, nil)
		if err != nil {
			return fmt.Errorf("error preparing television: %s", err)
		}

		err = fpstv.(gui.GUI).SetFeature(gui.ReqSetVisibility, true)
		if err != nil {
			return fmt.Errorf("error preparing television: %s", err)
		}
	} else {
		fpstv, err = television.NewBasicTelevision("NTSC")
		if err != nil {
			return fmt.Errorf("error preparing television: %s", err)
		}
	}

	vcs, err := hardware.NewVCS(fpstv)
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
	fn, err := fpstv.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}
	startFrame := fn

	// run for specified period of time

	// -- parse supplied duration
	duration, err := time.ParseDuration(runTime)
	if err != nil {
		return err
	}

	// -- setup trigger that expires when duration has elapsed
	var timerRunning atomic.Value
	timerRunning.Store(1)

	go func() {
		// force a two second leadtime to allow framerate to settle down
		time.AfterFunc(2*time.Second, func() {
			fn, err = fpstv.GetState(television.ReqFramenum)
			if err != nil {
				panic(err)
			}

			startFrame = fn

			time.AfterFunc(duration, func() {
				timerRunning.Store(-1)
			})
		})
	}()

	// -- run until specified time elapses (running is changed to -1)
	err = vcs.Run(func() (bool, error) {
		return timerRunning.Load().(int) > 0, nil
	})
	if err != nil {
		return err
	}

	// get ending frame number
	fn, err = vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}
	endFrame := fn

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
