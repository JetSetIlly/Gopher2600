package main

import (
	"flag"
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/debugger"
	"gopher2600/debugger/colorterm"
	"gopher2600/debugger/console"
	"gopher2600/disassembly"
	"gopher2600/gui"
	"gopher2600/gui/sdldebug"
	"gopher2600/gui/sdlplay"
	"gopher2600/magicflags"
	"gopher2600/paths"
	"gopher2600/performance"
	"gopher2600/playmode"
	"gopher2600/recorder"
	"gopher2600/regression"
	"gopher2600/television"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
)

const defaultInitScript = "debuggerInit"

func main() {
	// we generate random numbers in some places. seed the generator with the
	// current time
	rand.Seed(int64(time.Now().Second()))

	mf := magicflags.MagicFlags{
		ProgModes:   []string{"RUN", "PLAY", "DEBUG", "DISASM", "PERFORMANCE", "REGRESS"},
		DefaultMode: "RUN",
	}

	switch mf.Parse(os.Args[1:]) {
	case magicflags.ParseNoArgs:
		// no arguments at all. suggest that a cartridge is required
		fmt.Println("* 2600 cartridge required")
		os.Exit(2)
	case magicflags.ParseHelp:
		os.Exit(2)
	case magicflags.ParseContinue:
		break
	}

	ok := true

	switch mf.Mode {
	default:
		fmt.Printf("* %s mode unrecognised\n", mf.Mode)
		os.Exit(2)

	case "RUN":
		fallthrough

	case "PLAY":
		ok = play(&mf)

	case "DEBUG":
		ok = debug(&mf)

	case "DISASM":
		ok = disasm(&mf)

	case "PERFORMANCE":
		ok = perform(&mf)

	case "REGRESS":
		ok = regress(&mf)
	}

	if !ok {
		os.Exit(2)
	}
}

func play(mf *magicflags.MagicFlags) bool {
	cartFormat := mf.SubModeFlags.String("cartformat", "AUTO", "force use of cartridge format")
	tvType := mf.SubModeFlags.String("tv", "AUTO", "television specification: NTSC, PAL")
	scaling := mf.SubModeFlags.Float64("scale", 3.0, "television scaling")
	stable := mf.SubModeFlags.Bool("stable", true, "wait for stable frame before opening display")
	fpscap := mf.SubModeFlags.Bool("fpscap", true, "cap fps to specification")
	record := mf.SubModeFlags.Bool("record", false, "record user input to a file")

	if mf.SubParse() != magicflags.ParseContinue {
		return false
	}

	switch len(mf.SubModeFlags.Args()) {
	case 0:
		fmt.Println("* 2600 cartridge required")
		return false
	case 1:
		cartload := cartridgeloader.Loader{
			Filename: mf.SubModeFlags.Arg(0),
			Format:   *cartFormat,
		}

		tv, err := television.NewTelevision(*tvType)
		if err != nil {
			fmt.Printf("* %s\n", err)
			return false
		}

		scr, err := sdlplay.NewSdlPlay(tv, float32(*scaling))
		if err != nil {
			fmt.Printf("* %s\n", err)
			return false
		}

		err = playmode.Play(tv, scr, *stable, *fpscap, *record, cartload)
		if err != nil {
			fmt.Printf("* %s\n", err)
			return false
		}
		if *record {
			fmt.Println("! recording completed")
		}
	default:
		fmt.Printf("* too many arguments for %s mode\n", mf.Mode)
		return false
	}

	return true
}

func debug(mf *magicflags.MagicFlags) bool {
	cartFormat := mf.SubModeFlags.String("cartformat", "AUTO", "force use of cartridge format")
	tvType := mf.SubModeFlags.String("tv", "AUTO", "television specification: NTSC, PAL")
	termType := mf.SubModeFlags.String("term", "COLOR", "terminal type to use in debug mode: COLOR, PLAIN")
	initScript := mf.SubModeFlags.String("initscript", paths.ResourcePath(defaultInitScript), "terminal type to use in debug mode: COLOR, PLAIN")
	profile := mf.SubModeFlags.Bool("profile", false, "run debugger through cpu profiler")

	if mf.SubParse() != magicflags.ParseContinue {
		return false
	}

	tv, err := television.NewTelevision(*tvType)
	if err != nil {
		fmt.Printf("* %s\n", err)
		return false
	}

	scr, err := sdldebug.NewSdlDebug(tv, 2.0)
	if err != nil {
		fmt.Printf("* %s\n", err)
		return false
	}

	dbg, err := debugger.NewDebugger(tv, scr)
	if err != nil {
		fmt.Printf("* %s\n", err)
		return false
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
		term = &colorterm.ColorTerminal{}
	}

	switch len(mf.SubModeFlags.Args()) {
	case 0:
		// it's okay if DEBUG mode is started with no cartridges
		fallthrough
	case 1:
		runner := func() error {
			cartload := cartridgeloader.Loader{
				Filename: mf.SubModeFlags.Arg(0),
				Format:   *cartFormat,
			}
			err := dbg.Start(term, *initScript, cartload)
			if err != nil {
				return err
			}
			return nil
		}

		if *profile {
			err := performance.ProfileCPU("debug.cpu.profile", runner)
			if err != nil {
				fmt.Printf("* %s\n", err)
				return false
			}
			err = performance.ProfileMem("debug.mem.profile")
			if err != nil {
				fmt.Printf("* %s\n", err)
				return false
			}
		} else {
			err := runner()
			if err != nil {
				fmt.Printf("* %s\n", err)
				return false
			}
		}
	default:
		fmt.Printf("* too many arguments for %s mode\n", mf.Mode)
		return false
	}

	return true
}

func disasm(mf *magicflags.MagicFlags) bool {
	cartFormat := mf.SubModeFlags.String("cartformat", "AUTO", "force use of cartridge format")

	if mf.SubParse() != magicflags.ParseContinue {
		return false
	}

	switch len(mf.SubModeFlags.Args()) {
	case 0:
		fmt.Println("* 2600 cartridge required")
		return false
	case 1:
		cartload := cartridgeloader.Loader{
			Filename: mf.SubModeFlags.Arg(0),
			Format:   *cartFormat,
		}
		dsm, err := disassembly.FromCartrige(cartload)
		if err != nil {
			// print what disassembly output we do have
			if dsm != nil {
				dsm.Dump(os.Stdout)
			}

			// exit with error message
			fmt.Printf("* %s\n", err)
			return false
		}
		dsm.Dump(os.Stdout)
	default:
		fmt.Printf("* too many arguments for %s mode\n", mf.Mode)
		return false
	}

	return true
}

func perform(mf *magicflags.MagicFlags) bool {
	cartFormat := mf.SubModeFlags.String("cartformat", "AUTO", "force use of cartridge format")
	display := mf.SubModeFlags.Bool("display", false, "display TV output: boolean")
	fpscap := mf.SubModeFlags.Bool("fpscap", true, "cap FPS to specification (only valid if --display=true)")
	scaling := mf.SubModeFlags.Float64("scale", 3.0, "display scaling (only valid if --display=true")
	tvType := mf.SubModeFlags.String("tv", "AUTO", "television specification: NTSC, PAL")
	runTime := mf.SubModeFlags.String("time", "5s", "run duration (note: there is a 2s overhead)")
	profile := mf.SubModeFlags.Bool("profile", false, "perform cpu and memory profiling")

	if mf.SubParse() != magicflags.ParseContinue {
		return false
	}

	switch len(mf.SubModeFlags.Args()) {
	case 0:
		fmt.Println("* 2600 cartridge required")
		return false
	case 1:
		cartload := cartridgeloader.Loader{
			Filename: mf.SubModeFlags.Arg(0),
			Format:   *cartFormat,
		}

		tv, err := television.NewTelevision(*tvType)
		if err != nil {
			fmt.Printf("* %s\n", err)
			return false
		}

		if *display {
			scr, err := sdlplay.NewSdlPlay(tv, float32(*scaling))
			if err != nil {
				fmt.Printf("* %s\n", err)
				return false
			}

			err = scr.(gui.GUI).SetFeature(gui.ReqSetVisibility, true)
			if err != nil {
				fmt.Printf("* %s\n", err)
				return false
			}

			err = scr.(gui.GUI).SetFeature(gui.ReqSetFPSCap, *fpscap)
			if err != nil {
				fmt.Printf("* %s\n", err)
				return false
			}
		}

		err = performance.Check(os.Stdout, *profile, tv, *runTime, cartload)
		if err != nil {
			fmt.Printf("* %s\n", err)
			return false
		}
	default:
		fmt.Printf("* too many arguments for %s mode\n", mf.Mode)
		return false
	}

	return true
}

type yesReader struct{}

func (*yesReader) Read(p []byte) (n int, err error) {
	p[0] = 'y'
	return 1, nil
}

func regress(mf *magicflags.MagicFlags) bool {
	mf.SubMode = strings.ToUpper(mf.Next())
	mf.TryDefault()
	switch mf.SubMode {
	default:
		mf.ValidSubModes = []string{"RUN", "LIST", "DELETE", "ADD"}
		mf.DefaultSubMode = "RUN"

		if mf.SubParse() != magicflags.ParseContinue {
			return false
		}

		mf.DefaultFound()
		fallthrough

	case "RUN":
		// no additional arguments
		verbose := mf.SubModeFlags.Bool("verbose", false, "output more detail (eg. error messages)")
		failOnError := mf.SubModeFlags.Bool("fail", false, "fail on error")

		if mf.SubParse() != magicflags.ParseContinue {
			return false
		}

		err := regression.RegressRunTests(os.Stdout, *verbose, *failOnError, mf.SubModeFlags.Args())
		if err != nil {
			fmt.Printf("* %s\n", err)
			return false
		}

	case "LIST":
		// no additional arguments
		if mf.SubParse() != magicflags.ParseContinue {
			return false
		}
		switch len(mf.SubModeFlags.Args()) {
		case 0:
			err := regression.RegressList(os.Stdout)
			if err != nil {
				fmt.Printf("*  %s\n", err)
				return false
			}
		default:
			fmt.Printf("* no additional arguments required when using %s %s\n", mf.Mode, mf.SubMode)
			return false
		}

	case "DELETE":
		answerYes := mf.SubModeFlags.Bool("yes", false, "answer yes to confirmation")

		if mf.SubParse() != magicflags.ParseContinue {
			return false
		}

		switch len(mf.SubModeFlags.Args()) {
		case 0:
			fmt.Println("* database key required (use REGRESS LIST to view)")
			return false
		case 1:

			// use stdin for confirmation unless "yes" flag has been sent
			var confirmation io.Reader
			if *answerYes {
				confirmation = &yesReader{}
			} else {
				confirmation = os.Stdin
			}

			err := regression.RegressDelete(os.Stdout, confirmation, mf.SubModeFlags.Arg(0))
			if err != nil {
				fmt.Printf("* %s\n", err)
				return false
			}
		default:
			fmt.Printf("* only one entry can be deleted at at time when using %s %s\n", mf.Mode, mf.SubMode)
			return false
		}

	case "ADD":
		return regressAdd(mf)
	}

	return true
}

func regressAdd(mf *magicflags.MagicFlags) bool {
	cartFormat := mf.SubModeFlags.String("cartformat", "AUTO", "force use of cartridge format")
	tvType := mf.SubModeFlags.String("tv", "AUTO", "television specification: NTSC, PAL (cartridge args only)")
	numFrames := mf.SubModeFlags.Int("frames", 10, "number of frames to run (cartridge args only)")
	state := mf.SubModeFlags.Bool("state", false, "record TV state at every CPU step")
	notes := mf.SubModeFlags.String("notes", "", "annotation for the database")

	if mf.SubParse() != magicflags.ParseContinue {
		return false
	}

	switch len(mf.SubModeFlags.Args()) {
	case 0:
		fmt.Println("* 2600 cartridge or playback file required")
		return false
	case 1:
		var rec regression.Regressor

		if recorder.IsPlaybackFile(mf.SubModeFlags.Arg(0)) {
			// check and warn if unneeded arguments have been specified
			mf.SubModeFlags.Visit(func(flg *flag.Flag) {
				if flg.Name == "frames" {
					fmt.Printf("! ignored %s flag when adding playback entry\n", flg.Name)
				}
			})

			rec = &regression.PlaybackRegression{
				Script: mf.SubModeFlags.Arg(0),
				Notes:  *notes,
			}
		} else {
			cartload := cartridgeloader.Loader{
				Filename: mf.SubModeFlags.Arg(0),
				Format:   *cartFormat,
			}
			rec = &regression.FrameRegression{
				CartLoad:  cartload,
				TVtype:    strings.ToUpper(*tvType),
				NumFrames: *numFrames,
				State:     *state,
				Notes:     *notes,
			}
		}

		err := regression.RegressAdd(os.Stdout, rec)
		if err != nil {
			fmt.Printf("\r* error adding regression test: %s\n", err)
			return false
		}
	default:
		fmt.Printf("* regression tests must be added one at a time when using %s %s\n", mf.Mode, mf.SubMode)
		return false
	}

	return true
}
