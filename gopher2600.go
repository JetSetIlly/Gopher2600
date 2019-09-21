package main

import (
	"flag"
	"fmt"
	"gopher2600/debugger"
	"gopher2600/debugger/colorterm"
	"gopher2600/debugger/console"
	"gopher2600/disassembly"
	"gopher2600/hardware/memory"
	"gopher2600/performance"
	"gopher2600/playmode"
	"gopher2600/recorder"
	"gopher2600/regression"
	"io"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"
)

const defaultInitScript = ".gopher2600/debuggerInit"

func main() {
	// we generate random numbers in some places. seed the generator with the
	// current time
	rand.Seed(int64(time.Now().Second()))

	progName := path.Base(os.Args[0])

	var mode string
	var argList []string
	var argListPos int

	progModes := []string{"RUN", "PLAY", "DEBUG", "DISASM", "PERFORMANCE", "REGRESS"}
	defaultMode := "RUN"

	progFlags := flag.NewFlagSet(progName, flag.ContinueOnError)

	// we never want progFlags.Parse() to print out its own error messages
	progFlags.SetOutput(&nopWriter{})

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
		argList = os.Args[1:]
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
			argList = progFlags.Args()

			// ... unless it apears in the list of modes. in which case, the
			// single argument is a specified mode. let the mode switch below
			// handle what to do next.
			arg := strings.ToUpper(progFlags.Arg(0))
			for i := range progModes {
				if progModes[i] == arg {
					mode = arg
					argList = progFlags.Args()
					argListPos = 1
					break
				}
			}
		default:
			// many arguments have been supplied
			mode = strings.ToUpper(progFlags.Arg(0))
			argList = progFlags.Args()
			argListPos = 1
		}
	}

	// modes can have their own sets of flags
	usageBanner := strings.Join(progFlags.Args()[:len(progFlags.Args())-1], " ")
	usageBanner = strings.ToUpper(usageBanner)
	usageBanner = fmt.Sprintf("%s %s", progName, usageBanner)

	modeFlags := flag.NewFlagSet(usageBanner, flag.ContinueOnError)

	var subMode string
	var defaultSubMode string
	var validSubModes []string

	modeFlagsParse := func() {
		// return immediately if there are no more flags to parse
		if len(argList) < 1 || argListPos > len(argList) {
			return
		}

		// we don't want the regular -help message to be printed if a list of
		// sub-modes has been supplied
		if len(validSubModes) > 0 {
			modeFlags.SetOutput(&nopWriter{})
		}

		err := modeFlags.Parse(argList[argListPos:])
		if err != nil && err == flag.ErrHelp {
			if len(validSubModes) > 0 {
				fmt.Printf("available sub-modes for %s: %s\n", mode, strings.Join(validSubModes, ", "))
				if defaultSubMode != "" {
					fmt.Printf("default: %s\n", defaultSubMode)
				}
			}

			// error handling is less fancy than for progFlag parsing. the
			// default sub-modes can be handled by a fallthrough

			os.Exit(2)
		}
	}

	switch mode {
	default:
		fmt.Printf("* %s mode unrecognised\n", mode)
		os.Exit(2)

	case "RUN":
		fallthrough

	case "PLAY":
		cartFormat := modeFlags.String("cartformat", "AUTO", "force use of cartridge format")
		tvType := modeFlags.String("tv", "AUTO", "television specification: NTSC, PAL")
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
			cartload := memory.CartridgeLoader{
				Filename: modeFlags.Arg(0),
				Format:   *cartFormat,
			}

			err := playmode.Play(*tvType, float32(*scaling), *stable, *recording, *record, cartload)
			if err != nil {
				fmt.Printf("* %s\n", err)
				os.Exit(2)
			}
			if *record {
				fmt.Println("! recording completed")
			} else if *recording != "" {
				fmt.Println("! playback completed")
			}
		default:
			fmt.Printf("* too many arguments for %s mode\n", mode)
			os.Exit(2)
		}

	case "DEBUG":
		cartFormat := modeFlags.String("cartformat", "AUTO", "force use of cartridge format")
		tvType := modeFlags.String("tv", "AUTO", "television specification: NTSC, PAL")
		termType := modeFlags.String("term", "COLOR", "terminal type to use in debug mode: COLOR, PLAIN")
		initScript := modeFlags.String("initscript", defaultInitScript, "terminal type to use in debug mode: COLOR, PLAIN")
		profile := modeFlags.Bool("profile", false, "run debugger through cpu profiler")
		modeFlagsParse()

		dbg, err := debugger.NewDebugger(*tvType)
		if err != nil {
			fmt.Printf("* %s\n", err)
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
			runner := func() error {
				cartload := memory.CartridgeLoader{
					Filename: modeFlags.Arg(0),
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
					os.Exit(2)
				}
				err = performance.ProfileMem("debug.mem.profile")
				if err != nil {
					fmt.Printf("* %s\n", err)
					os.Exit(2)
				}
			} else {
				err := runner()
				if err != nil {
					fmt.Printf("* %s\n", err)
					os.Exit(2)
				}
			}
		default:
			fmt.Printf("* too many arguments for %s mode\n", mode)
			os.Exit(2)
		}

	case "DISASM":
		cartFormat := modeFlags.String("cartformat", "AUTO", "force use of cartridge format")
		modeFlagsParse()

		switch len(modeFlags.Args()) {
		case 0:
			fmt.Println("* 2600 cartridge required")
			os.Exit(2)
		case 1:
			cartload := memory.CartridgeLoader{
				Filename: modeFlags.Arg(0),
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
				os.Exit(2)
			}
			dsm.Dump(os.Stdout)
		default:
			fmt.Printf("* too many arguments for %s mode\n", mode)
			os.Exit(2)
		}

	case "PERFORMANCE":
		cartFormat := modeFlags.String("cartformat", "AUTO", "force use of cartridge format")
		display := modeFlags.Bool("display", false, "display TV output: boolean")
		tvType := modeFlags.String("tv", "AUTO", "television specification: NTSC, PAL")
		scaling := modeFlags.Float64("scale", 3.0, "television scaling")
		runTime := modeFlags.String("time", "5s", "run duration (note: there is a 2s overhead)")
		profile := modeFlags.Bool("profile", false, "perform cpu and memory profiling")
		modeFlagsParse()

		switch len(modeFlags.Args()) {
		case 0:
			fmt.Println("* 2600 cartridge required")
			os.Exit(2)
		case 1:
			cartload := memory.CartridgeLoader{
				Filename: modeFlags.Arg(0),
				Format:   *cartFormat,
			}
			err := performance.Check(os.Stdout, *profile, *display, *tvType, float32(*scaling), *runTime, cartload)
			if err != nil {
				fmt.Printf("* %s\n", err)
				os.Exit(2)
			}
		default:
			fmt.Printf("* too many arguments for %s mode\n", mode)
			os.Exit(2)
		}

	case "REGRESS":
		subMode = strings.ToUpper(progFlags.Arg(1))
		argListPos++
		switch subMode {
		default:
			validSubModes = []string{"RUN", "LIST", "DELETE", "ADD"}
			defaultSubMode = "RUN"
			modeFlagsParse()
			argListPos-- // undo modeArgPos adjustment
			fallthrough

		case "RUN":
			// no additional arguments
			verbose := modeFlags.Bool("verbose", false, "output more detail (eg. error messages)")
			failOnError := modeFlags.Bool("fail", false, "fail on error")
			modeFlagsParse()

			err := regression.RegressRunTests(os.Stdout, *verbose, *failOnError, modeFlags.Args())
			if err != nil {
				fmt.Printf("* %s\n", err)
				os.Exit(2)
			}

		case "LIST":
			// no additional arguments
			modeFlagsParse()
			switch len(modeFlags.Args()) {
			case 0:
				err := regression.RegressList(os.Stdout)
				if err != nil {
					fmt.Printf("*  %s\n", err)
					os.Exit(2)
				}
			default:
				fmt.Printf("* no additional arguments required when using %s %s\n", mode, subMode)
				os.Exit(2)
			}

		case "DELETE":
			answerYes := modeFlags.Bool("yes", false, "answer yes to confirmation")
			modeFlagsParse()

			switch len(modeFlags.Args()) {
			case 0:
				fmt.Println("* database key required (use REGRESS LIST to view)")
				os.Exit(2)
			case 1:

				// use stdin for confirmation unless "yes" flag has been sent
				var confirmation io.Reader
				if *answerYes {
					confirmation = new(yesReader)
				} else {
					confirmation = os.Stdin
				}

				err := regression.RegressDelete(os.Stdout, confirmation, modeFlags.Arg(0))
				if err != nil {
					fmt.Printf("* %s\n", err)
					os.Exit(2)
				}
			default:
				fmt.Printf("* only one entry can be deleted at at time when using %s %s\n", mode, subMode)
				os.Exit(2)
			}

		case "ADD":
			cartFormat := modeFlags.String("cartformat", "AUTO", "force use of cartridge format")
			tvType := modeFlags.String("tv", "AUTO", "television specification: NTSC, PAL (cartridge args only)")
			numFrames := modeFlags.Int("frames", 10, "number of frames to run (cartridge args only)")
			state := modeFlags.Bool("state", false, "record TV state at every CPU step")
			notes := modeFlags.String("notes", "", "annotation for the database")
			modeFlagsParse()

			switch len(modeFlags.Args()) {
			case 0:
				fmt.Println("* 2600 cartridge or playback file required")
				os.Exit(2)
			case 1:
				var rec regression.Regressor

				if recorder.IsPlaybackFile(modeFlags.Arg(0)) {
					// check and warn if unneeded arguments have been specified
					modeFlags.Visit(func(flg *flag.Flag) {
						if flg.Name == "frames" {
							fmt.Printf("! ignored %s flag when adding playback entry\n", flg.Name)
						}
					})

					rec = &regression.PlaybackRegression{
						Script: modeFlags.Arg(0),
						Notes:  *notes,
					}
				} else {
					cartload := memory.CartridgeLoader{
						Filename: modeFlags.Arg(0),
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
					os.Exit(2)
				}
			default:
				fmt.Printf("* regression tests must be added one at a time when using %s %s\n", mode, subMode)
				os.Exit(2)
			}
		}
	}
}

// special purpose io.Reader / io.Writer

type nopWriter struct{}

func (*nopWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

type yesReader struct{}

func (*yesReader) Read(p []byte) (n int, err error) {
	p[0] = 'y'
	return 1, nil
}
