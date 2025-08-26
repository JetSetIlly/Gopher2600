// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/colorterm"
	"github.com/jetsetilly/gopher2600/debugger/terminal/plainterm"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/performance"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/regression"
	"github.com/jetsetilly/gopher2600/resources"
	"github.com/jetsetilly/gopher2600/version"
)

// communication between the main goroutine and the launch goroutine.
type mainSync struct {
	state chan stateRequest

	// a created GUI will communicate thought these channels
	gui      chan guiControl
	guiError chan error
}

// the stateRequest sent through the state channel in mainSync.
type stateReq string

// list of valid stateReq values.
const (
	// main thread should end as soon as possible.
	//
	// takes optional int argument, indicating the status code.
	reqQuit stateReq = "QUIT"

	// reset interrupt signal handling. used when an alternative
	// handler is more appropriate. for example, the playMode and Debugger
	// package provide a mode specific handler.
	//
	// takes no arguments.
	reqNoIntSig stateReq = "NOINTSIG"

	// the gui creation function to run in the main goroutine. this is for GUIs
	// that *need* to be run in the main OS thead (SDL, etc.)
	//
	// the only argument must be a guiCreate reference.
	reqCreateGUI stateReq = "CREATEGUI"
)

type stateRequest struct {
	req  stateReq
	args any
}

// the gui create function. paired with reqCreateGUI state request.
type guiCreate func() (guiControl, error)

// guiControl defines the functions that a guiControl implementation must implement to be
// usable from the main goroutine.
type guiControl interface {
	// cleanup resources used by the gui
	Destroy()

	// Service() should not pause or loop longer than necessary (if at all). It
	// MUST ONLY by called as part of a larger loop from the main thread. It
	// should service all gui events that are not safe to do in sub-threads.
	//
	// If the GUI framework does not require this sort of thread safety then
	// there is no need for the Service() function to do anything.
	Service()
}

func main() {
	sync := &mainSync{
		state:    make(chan stateRequest),
		gui:      make(chan guiControl),
		guiError: make(chan error),
	}

	// the value to use with os.Exit(). can be changed with reqQuit
	// stateRequest
	exitVal := 0

	// ctrlc default handler. can be turned off with reqNoIntSig request
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt)

	// launch program as a go routine. further communication is through
	// the mainSync instance
	go launch(sync, os.Args[1:])

	// if there is no GUI then we should sleep so that the select channel loop
	// doesn't go beserk
	noGuiSleepPeriod, err := time.ParseDuration("5ms")
	if err != nil {
		panic(err)
	}

	// loop until done is true. every iteration of the loop we listen for:
	//
	//  1. interrupt signals
	//  2. new gui creation functions
	//  3. state requests
	//  4. (default) anything in the Service() function of the most recently created GUI
	//
	done := false
	var gui guiControl
	for !done {
		select {
		case <-intChan:
			fmt.Println("\r")
			done = true

		case state := <-sync.state:
			switch state.req {
			case reqQuit:
				done = true
				if gui != nil {
					gui.Destroy()
				}

				if state.args != nil {
					if v, ok := state.args.(int); ok {
						exitVal = v
					} else {
						panic(fmt.Sprintf("cannot convert %s arguments into int", reqQuit))
					}
				}

			case reqNoIntSig:
				signal.Reset(os.Interrupt)
				if state.args != nil {
					panic(fmt.Sprintf("%s does not accept any arguments", reqNoIntSig))
				}

			case reqCreateGUI:
				var err error

				// destroy existing gui
				if gui != nil {
					gui.Destroy()
				}

				gui, err = state.args.(guiCreate)()
				if err != nil {
					sync.guiError <- err

					// gui is a variable of type interface. nil doesn't work as you
					// might expect with interfaces. for instance, even though the
					// following outputs "<nil>":
					//
					//	fmt.Println(gui)
					//
					// the following equation print false:
					//
					//	fmt.Println(gui == nil)
					//
					// as to the reason why gui does not equal nil, even though
					// the creator() function returns nil? well, you tell me.
					gui = nil
				} else {
					sync.gui <- gui
				}
			}

		default:
			// if an instance of gui.Events has been sent to us via sync.events
			// then call Service(). otherwise, sleep for a very short period
			if gui != nil {
				gui.Service()
			} else {
				time.Sleep(noGuiSleepPeriod)
			}
		}
	}

	fmt.Print("\r")
	os.Exit(exitVal)
}

// launch is called from main() as a goroutine. uses mainSync instance to
// indicate gui creation and to quit.
func launch(sync *mainSync, args []string) {
	// log version
	ver, rev, _ := version.Version()
	logger.Logf(logger.Allow, "gopher2600", "%s", ver)
	logger.Logf(logger.Allow, "gopher2600", "%s", rev)

	// number of cores
	logger.Logf(logger.Allow, "gopher2600", "number of cores being used: %d", runtime.NumCPU())

	// use flag set to provide the --help flag for top level command line.
	// that's all we want it to do
	flgs := flag.NewFlagSet(version.ApplicationName, flag.ContinueOnError)

	// setting flag output to the nilWriter because we need to control how
	// unrecognised arguments are displayed
	flgs.SetOutput(&nilWriter{})

	// parse arguments. if the help flag has been used then print out the
	// execution modes summary and return
	err := flgs.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flgs.Usage()
			fmt.Println("Execution Modes: RUN, DEBUG, DISASM, PERFORMANCE, REGRESS, VERSION")
			sync.state <- stateRequest{req: reqQuit, args: 0}
			return
		}

		// ignoring any other flag.Parse() error. this can happen when an
		// argument is intended for the default run mode
	} else {
		// get remaining arguments for passing to execution mode functions
		args = flgs.Args()
	}

	// get mode from command line
	var mode string

	if len(args) > 0 {
		mode = strings.ToUpper(args[0])
	}

	// switch on execution modes
	switch mode {
	default:
		mode = "RUN"
		err = emulate(mode, sync, args)
	case "RUN":
		fallthrough
	case "PLAY":
		fallthrough
	case "DEBUG":
		err = emulate(mode, sync, args[1:])
	case "DISASM":
		err = disasm(mode, args[1:])
	case "PERFORMANCE":
		err = perform(mode, sync, args[1:])
	case "REGRESS":
		err = regress(mode, args[1:])
	case "VERSION":
		err = showVersion(mode, args[1:])
	}

	if err != nil {
		// swallow power off error messages. send quit signal with return value of 20 instead
		if !errors.Is(err, ports.PowerOff) {
			fmt.Printf("* error in %s mode: %s\n", mode, err)
			sync.state <- stateRequest{req: reqQuit, args: 20}
			return
		}
	}

	sync.state <- stateRequest{req: reqQuit, args: 0}
}

const defaultInitScript = "debuggerInit"

// emulate is the main emulation launch function, shared by play and debug
// modes. the other modes initialise and run the emulation differently.
func emulate(mode string, sync *mainSync, args []string) error {
	var emulationMode govern.Mode
	switch mode {
	case "PLAY":
		emulationMode = govern.ModePlay
	case "RUN":
		emulationMode = govern.ModePlay
	case "DEBUG":
		emulationMode = govern.ModeDebugger
	default:
		panic(fmt.Errorf("unknown emulation mode: %s", mode))
	}

	// opts collates the individual options that can be set by the command line
	var opts debugger.CommandLineOptions

	// arguments common to both play and debugging modes
	flgs := flag.NewFlagSet(mode, flag.ExitOnError)
	flgs.BoolVar(&opts.Log, "log", false, "echo debugging log to stdout")
	flgs.StringVar(&opts.Spec, "tv", "AUTO",
		fmt.Sprintf("television specification: %s", strings.Join(specification.ReqSpecList, ", ")))
	flgs.BoolVar(&opts.FpsCap, "fpscap", true, "cap FPS to emulation TV")
	flgs.IntVar(&opts.Multiload, "multiload", -1, "force multiload byte (supercharger only; 0 to 255")
	flgs.StringVar(&opts.Mapping, "mapping", "AUTO", "force cartridge mapper selection")
	flgs.StringVar(&opts.Bank, "bank", "AUTO", "selected cartridge bank on reset")
	flgs.StringVar(&opts.Left, "left", "AUTO", "left player port: AUTO, STICK, PADDLE, KEYPAD, GAMEPAD")
	flgs.StringVar(&opts.Right, "right", "AUTO", "left player port: AUTO, STICK, PADDLE, KEYPAD, GAMEPAD")
	flgs.BoolVar(&opts.SwapPorts, "swap", false, "swap player ports")
	flgs.StringVar(&opts.Profile, "profile", "none", "run performance check with profiling: CPU, MEM, TRACE, ALL (comma sep)")
	flgs.StringVar(&opts.DWARF, "dwarf", "", "path to DWARF file. only valid for some coproc supporting ROMs")

	// playmode specific arguments
	if emulationMode == govern.ModePlay {
		flgs.StringVar(&opts.ComparisonROM, "comparisonROM", "", "ROM to run in parallel for comparison")
		flgs.StringVar(&opts.ComparisonPrefs, "comparisonPrefs", "", "preferences for comparison emulation")
		flgs.BoolVar(&opts.Record, "record", false, "record user input to new file for future playback")
		flgs.StringVar(&opts.RecordFilename, "recordFilename", "", "set output name for recording")
		flgs.BoolVar(&opts.PlaybackCheckROM, "playbackCheckROM", true, "check ROM hash on playback")
		flgs.BoolVar(&opts.PlaybackIgnoreDigest, "playbackIgnoreDigest", false, "ignore video digests in playback files")
		flgs.StringVar(&opts.PatchFile, "patch", "", "patch to apply to emulation (not playback files)")
		flgs.BoolVar(&opts.Wav, "wav", false, "record audio to wav file")
		flgs.BoolVar(&opts.Video, "video", false, "record video to mp4 file")
		flgs.BoolVar(&opts.NoEject, "noeject", false, "emulator will not quit is noeject is true")
		flgs.StringVar(&opts.Macro, "macro", "", "macro file to be run on trigger")
	}

	// debugger specific arguments
	if emulationMode == govern.ModeDebugger {
		// prepare the path to the initialisation script used by the debugger. we
		// can name the file in the defaultInitScript const declaration but the
		// construction of the path is platform sensitive so we must do it here
		defInitScript, err := resources.JoinPath(defaultInitScript)
		if err != nil {
			return err
		}

		flgs.StringVar(&opts.InitScript, "initscript", defInitScript, "script to run on debugger start")
		flgs.StringVar(&opts.TermType, "term", "IMGUI", "terminal type: IMGUI, COLOR, PLAIN")
	} else {
		// non debugger emulation is always of type IMGUI
		opts.TermType = "IMGUI"
	}

	// parse args and get copy of remaining arguments
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	args = flgs.Args()

	// check remaining arguments. if there are any outstanding arguments to
	// process then the user has made a mistake
	if len(args) > 1 {
		return fmt.Errorf("too many arguments")
	}

	// turn logging on by setting the echo function. events are still logged
	// and available via the debugger but will not be "echoed" to the terminal,
	// unless this option is on
	if opts.Log {
		logger.SetEcho(os.Stdout, true)
	} else {
		logger.SetEcho(nil, false)
	}

	// turn off fallback ctrl-c handling. this so that the debugger can handle
	// quit events more gracefully
	//
	// we must do this before creating the emulation or we'll just end up
	// turning the emulation's interrupt handler off
	sync.state <- stateRequest{req: reqNoIntSig}

	// prepare new debugger, supplying a debugger.CreateUserInterface function.
	// this function will be called by NewDebugger() and in turn will send a
	// GUI create message to the main goroutine
	dbg, err := debugger.NewDebugger(opts, func(e *debugger.Debugger) (gui.GUI, terminal.Terminal, error) {
		var term terminal.Terminal
		var scr gui.GUI

		// create GUI as appropriate
		if opts.TermType == "IMGUI" {
			sync.state <- stateRequest{req: reqCreateGUI,
				args: guiCreate(func() (guiControl, error) {
					return sdlimgui.NewSdlImgui(e)
				}),
			}

			// wait for creator result
			select {
			case g := <-sync.gui:
				scr = g.(gui.GUI)
			case err := <-sync.guiError:
				return nil, nil, err
			}

			// if gui implements the terminal.Broker interface use that terminal
			// as a preference
			if b, ok := scr.(terminal.Broker); ok {
				term = b.GetTerminal()
			}
		} else {
			// no GUI specified so we use a stub
			scr = gui.Stub{}
		}

		// if the GUI does not supply a terminal then use a color or plain terminal
		// as a fallback
		if term == nil {
			switch strings.ToUpper(opts.TermType) {
			default:
				logger.Logf(logger.Allow, "terminal", "unknown terminal: %s", opts.TermType)
				logger.Log(logger.Allow, "terminal", "defaulting to plain")
				term = &plainterm.PlainTerminal{}
			case "PLAIN":
				term = &plainterm.PlainTerminal{}
			case "COLOR":
				term = &colorterm.ColorTerminal{}
			}
		}

		return scr, term, nil
	})
	if err != nil {
		return err
	}

	// set up a launch function. this function is called either directly or via
	// a call to performance.RunProfiler()
	dbgLaunch := func() error {
		var romFile string
		if len(args) != 0 {
			romFile = args[0]
		}

		switch emulationMode {
		case govern.ModeDebugger:
			err := dbg.StartInDebugMode(romFile)
			if err != nil {
				return err
			}

		case govern.ModePlay:
			err := dbg.StartInPlayMode(romFile)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// check for profiling option and either run the launch function (prepared
	// above) via the performance.RunProfiler() function or directly
	prf, err := performance.ParseProfileString(opts.Profile)
	if err != nil {
		return err
	}

	if prf == performance.ProfileNone {
		// no profile required so run dbgLaunch() function as normal
		err := dbgLaunch()
		if err != nil {
			return err
		}
	} else {
		// filename argument for RunProfiler
		s := ""
		switch emulationMode {
		case govern.ModeDebugger:
			s = "debugger"
		case govern.ModePlay:
			s = "play"
		}

		// if profile generation has been requested then pass the dbgLaunch()
		// function prepared above, through the RunProfiler() function
		err := performance.RunProfiler(prf, s, dbgLaunch)
		if err != nil {
			return err
		}
	}

	return nil
}

func disasm(mode string, args []string) error {
	var mapping string
	var bytecode bool
	var bank int

	flgs := flag.NewFlagSet(mode, flag.ExitOnError)
	flgs.StringVar(&mapping, "mapping", "AUTO", "force cartridge mapper selection")
	flgs.BoolVar(&bytecode, "bytecode", false, "including bytecode in disassembly")
	flgs.IntVar(&bank, "bank", -1, "show disassembly for a specific bank")

	// parse args and get copy of remaining arguments
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	args = flgs.Args()

	switch len(args) {
	case 0:
		return fmt.Errorf("2600 cartridge required")
	case 1:
		attr := disassembly.ColumnAttr{
			ByteCode: bytecode,
			Label:    true,
			Cycles:   true,
		}

		cartload, err := cartridgeloader.NewLoaderFromFilename(args[0], mapping, "AUTO", nil)
		if err != nil {
			return err
		}
		defer cartload.Close()

		dsm, err := disassembly.FromCartridge(cartload)
		if err != nil {
			// print what disassembly output we do have
			if dsm != nil {
				// ignore any further errors
				_ = dsm.Write(os.Stdout, attr)
			}
			return err
		}

		// output entire disassembly or just a specific bank
		if bank < 0 {
			err = dsm.Write(os.Stdout, attr)
		} else {
			err = dsm.WriteBank(os.Stdout, attr, bank)
		}

		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("too many arguments")
	}

	return nil
}

func perform(mode string, sync *mainSync, args []string) error {
	var mapping string
	var bank string
	var spec string
	var uncapped bool
	var duration string
	var profile string
	var log bool

	flgs := flag.NewFlagSet(mode, flag.ExitOnError)
	flgs.StringVar(&mapping, "mapping", "AUTO", "form cartridge mapper selection")
	flgs.StringVar(&bank, "bank", "AUTO", "selected cartridge bank on reset")
	flgs.StringVar(&spec, "tv", "AUTO",
		fmt.Sprintf("television specification: %s", strings.Join(specification.ReqSpecList, ", ")))
	flgs.BoolVar(&uncapped, "uncapped", true, "run performance no FPS cap")
	flgs.StringVar(&duration, "duration", "5s", "run duation (with an additional 2s overhead)")
	flgs.StringVar(&profile, "profile", "none", "run performance check with profiling: CPU, MEM, TRACE, ALL (comma sep)")
	flgs.BoolVar(&log, "log", false, "echo debugging log to stdout")

	// parse args and get copy of remaining arguments
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	args = flgs.Args()

	// set debugging log echo
	if log {
		logger.SetEcho(os.Stdout, true)
	} else {
		logger.SetEcho(nil, false)
	}

	switch len(args) {
	case 0:
		return fmt.Errorf("2600 cartridge required")
	case 1:
		cartload, err := cartridgeloader.NewLoaderFromFilename(args[0], mapping, bank, nil)
		if err != nil {
			return err
		}
		defer cartload.Close()

		// check for profiling options
		p, err := performance.ParseProfileString(profile)
		if err != nil {
			return err
		}

		// run performance check
		err = performance.Check(os.Stdout, p, cartload, spec, uncapped, duration)
		if err != nil {
			return err
		}

		// deliberately not saving gui preferences because we don't want any
		// changes to the performance window impacting the play mode

	default:
		return fmt.Errorf("too many arguments")
	}

	return nil
}

func regress(mode string, args []string) error {
	// use flag set to provide the --help flag
	flgs := flag.NewFlagSet(mode, flag.ContinueOnError)

	err := flgs.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Println("Sub modes: RUN, LIST, DELETE, ADD, REDUX, CLEANUP")
			return nil
		}
		return err
	} else {
		args = flgs.Args()
	}

	var subMode string

	if len(args) > 0 {
		subMode = strings.ToUpper(args[0])
	}

	switch subMode {
	default:
		err = regressRun(fmt.Sprintf("%s %s", mode, "RUN"), args)
	case "RUN":
		err = regressRun(fmt.Sprintf("%s %s", mode, subMode), args[1:])
	case "LIST":
		err = regressList(fmt.Sprintf("%s %s", mode, subMode), args[1:])
	case "DELETE":
		err = regressDelete(fmt.Sprintf("%s %s", mode, subMode), args[1:])
	case "ADD":
		err = regressAdd(fmt.Sprintf("%s %s", mode, subMode), args[1:])
	case "REDUX":
		err = regressRedux(fmt.Sprintf("%s %s", mode, subMode), args[1:])
	case "CLEANUP":
		err = regressCleanup(fmt.Sprintf("%s %s", mode, subMode), args[1:])
	}

	if err != nil {
		return err
	}

	return nil
}

func regressRun(mode string, args []string) error {
	var opts regression.RegressRunOptions

	flgs := flag.NewFlagSet(mode, flag.ExitOnError)
	flgs.BoolVar(&opts.Verbose, "verbose", false, "output more detail")
	flgs.BoolVar(&opts.Concurrent, "concurrent", true, "run tests concurrently where possible")

	// parse args and get copy of remaining arguments
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	opts.Keys = flgs.Args()

	err = regression.RegressRun(os.Stdout, opts)
	if err != nil {
		return err
	}

	return nil
}

func regressList(mode string, args []string) error {
	flgs := flag.NewFlagSet(mode, flag.ExitOnError)

	// parse args and get copy of remaining arguments
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	args = flgs.Args()

	err = regression.RegressList(os.Stdout, args)
	if err != nil {
		return err
	}

	return nil
}

func regressDelete(mode string, args []string) error {
	var yes bool

	flgs := flag.NewFlagSet(mode, flag.ExitOnError)
	flgs.BoolVar(&yes, "yes", false, "answer yes to confirmation request")

	// parse args and get copy of remaining arguments
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	args = flgs.Args()

	switch len(args) {
	case 0:
		return fmt.Errorf("database key required")
	case 1:
		// use stdin for confirmation unless "yes" flag has been sent
		var confirmation io.Reader
		if yes {
			confirmation = &yesReader{}
		} else {
			confirmation = os.Stdin
		}

		err := regression.RegressDelete(os.Stdout, confirmation, args[0])
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("only one entry can be deleted at at time")
	}

	return nil
}

func regressAdd(mode string, args []string) error {
	var regressMode string
	var notes string
	var mapping string
	var spec string
	var numFrames int
	var state string
	var log bool

	flgs := flag.NewFlagSet(mode, flag.ContinueOnError)
	flgs.StringVar(&regressMode, "mode", "", "type of regression entry")
	flgs.StringVar(&notes, "notes", "", "additional annotation for the entry")
	flgs.StringVar(&mapping, "mapping", "AUTO", "form cartridge mapper selection")
	flgs.StringVar(&spec, "tv", "AUTO",
		fmt.Sprintf("television specification: %s", strings.Join(specification.ReqSpecList, ", ")))
	flgs.IntVar(&numFrames, "frames", 10, "number of frames to run [not playback files]")
	flgs.StringVar(&state, "state", "", "record emulator state at every CPU step [not playback files]")
	flgs.BoolVar(&log, "log", false, "echo debugging log to stdout")

	// parse args and get copy of remaining arguments
	err := flgs.Parse(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Println()
			fmt.Println(`The regression test to be added can be the path to a cartridge file or a previously
recorded playback file. For playback files, the flags marked [not playback files] do not
make sense and will be ignored.

Available modes are VIDEO, PLAYBACK and LOG. If not mode is explicitly given then
VIDEO will be used for ROM files and PLAYBACK will be used for playback recordings.

Value for the -state flag can be one of TV, PORTS, TIMER, CPU and can be used
with the default VIDEO mode.

The -log flag intructs the program to echo the log to the console. Do not confuse this
with the LOG mode. Note that asking for log output will suppress regression progress meters.`)
			return nil
		}
		return err
	}
	args = flgs.Args()

	// set debugging log echo
	if log {
		logger.SetEcho(os.Stdout, true)
	} else {
		logger.SetEcho(nil, false)
	}

	switch len(args) {
	case 0:
		return fmt.Errorf("2600 cartridge or playback file required")
	case 1:
		var regressor regression.Regressor

		if regressMode == "" {
			if err := recorder.IsPlaybackFile(args[0]); err == nil {
				regressMode = "PLAYBACK"
			} else if !errors.Is(err, recorder.NotAPlaybackFile) {
				return err
			} else {
				regressMode = "VIDEO"
			}
		}

		switch strings.ToUpper(regressMode) {
		case "VIDEO":
			statetype, err := regression.NewStateType(state)
			if err != nil {
				return err
			}

			regressor = &regression.VideoRegression{
				Cartridge: args[0],
				Mapping:   mapping,
				TVtype:    strings.ToUpper(spec),
				NumFrames: numFrames,
				State:     statetype,
				Notes:     notes,
			}
		case "PLAYBACK":
			// check and warn if unneeded arguments have been specified

			regressor = &regression.PlaybackRegression{
				Script: args[0],
				Notes:  notes,
			}
		case "LOG":
			regressor = &regression.LogRegression{
				Cartridge: args[0],
				Mapping:   mapping,
				TVtype:    strings.ToUpper(spec),
				NumFrames: numFrames,
				Notes:     notes,
			}
		}

		err := regression.RegressAdd(os.Stdout, regressor)
		if err != nil {
			// using carriage return (without newline) at beginning of error
			// message because we want to overwrite the last output from
			// RegressAdd()
			return fmt.Errorf("\rerror adding regression test: %w", err)
		}
	default:
		return fmt.Errorf("regression tests can only be added one at a time")
	}

	return nil
}

func regressRedux(mode string, args []string) error {
	var yes bool
	var verbose bool

	flgs := flag.NewFlagSet(mode, flag.ExitOnError)
	flgs.BoolVar(&yes, "yes", false, "answer yes to confirmation request")
	flgs.BoolVar(&verbose, "v", false, "output more detail")

	// parse args and get copy of remaining arguments
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	args = flgs.Args()

	var confirmation io.Reader
	if yes {
		confirmation = &yesReader{}
	} else {
		confirmation = os.Stdin
	}

	return regression.RegressRedux(os.Stdout, confirmation, verbose, args)
}

func regressCleanup(mode string, args []string) error {
	var yes bool

	flgs := flag.NewFlagSet(mode, flag.ExitOnError)
	flgs.BoolVar(&yes, "yes", false, "answer yes to confirmation request")

	// parse args and get copy of remaining arguments
	err := flgs.Parse(args)
	if err != nil {
		return err
	}
	args = flgs.Args()

	var confirmation io.Reader
	if yes {
		confirmation = &yesReader{}
	} else {
		confirmation = os.Stdin
	}

	return regression.RegressCleanup(os.Stdout, confirmation)
}

func showVersion(mode string, args []string) error {
	var revision bool

	flgs := flag.NewFlagSet(mode, flag.ExitOnError)
	flgs.BoolVar(&revision, "v", false, "display revision information (if available")
	flgs.Parse(args)

	ver, rev, _ := version.Version()
	fmt.Println(ver)
	if revision {
		fmt.Println(rev)
	}

	return nil
}

// nilWriter is an empty writer.
type nilWriter struct{}

func (*nilWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

// yesReader always returns 'y' when it is read.
type yesReader struct{}

func (*yesReader) Read(p []byte) (n int, err error) {
	p[0] = 'y'
	return 1, nil
}
