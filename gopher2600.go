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
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/colorterm"
	"github.com/jetsetilly/gopher2600/debugger/terminal/plainterm"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/modalflag"
	"github.com/jetsetilly/gopher2600/performance"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/regression"
	"github.com/jetsetilly/gopher2600/resources"
	"github.com/jetsetilly/gopher2600/statsview"
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
	args interface{}
}

// the gui create function. paired with reqCreateGUI state request.
type guiCreate func() (guiControl, error)

// guiControl defines the functions that a guiControl implementation must implement to be
// usable from the main goroutine.
type guiControl interface {
	// cleanup resources used by the gui
	Destroy(io.Writer)

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

	// #ctrlc default handler. can be turned off with reqNoIntSig request
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt)

	// launch program as a go routine. further communication is  through
	// the mainSync instance
	go launch(sync)

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
	//  3. anything in the Service() function of the most recently created GUI
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
					gui.Destroy(os.Stderr)
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
					gui.Destroy(os.Stderr)
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
			// then call Service()
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
func launch(sync *mainSync) {
	// we generate random numbers in some places. seed the generator with the
	// current time
	rand.Seed(int64(time.Now().Nanosecond()))

	md := &modalflag.Modes{Output: os.Stdout}
	md.NewArgs(os.Args[1:])
	md.NewMode()
	md.AddSubModes("RUN", "PLAY", "DEBUG", "DISASM", "PERFORMANCE", "REGRESS")

	p, err := md.Parse()
	switch p {
	case modalflag.ParseHelp:
		sync.state <- stateRequest{req: reqQuit}
		return

	case modalflag.ParseError:
		fmt.Printf("* error: %v\n", err)
		// 10
		sync.state <- stateRequest{req: reqQuit, args: 10}
		return
	}

	switch md.Mode() {
	case "RUN":
		fallthrough

	case "PLAY":
		err = emulate(emulation.ModePlay, md, sync)

	case "DEBUG":
		err = emulate(emulation.ModeDebugger, md, sync)

	case "DISASM":
		err = disasm(md)

	case "PERFORMANCE":
		err = perform(md, sync)

	case "REGRESS":
		err = regress(md, sync)
	}

	if err != nil {
		// swallow power off error messages
		if !curated.Has(err, ports.PowerOff) {
			fmt.Printf("* error in %s mode: %s\n", md.String(), err)
			sync.state <- stateRequest{req: reqQuit, args: 20}
			return
		}
	}

	sync.state <- stateRequest{req: reqQuit}
}

const defaultInitScript = "debuggerInit"

func emulate(emulationMode emulation.Mode, md *modalflag.Modes, sync *mainSync) error {
	md.NewMode()

	defInitScript, err := resources.JoinPath(defaultInitScript)
	if err != nil {
		return err
	}

	mapping := md.AddString("mapping", "AUTO", "force use of cartridge mapping")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL, PAL60")
	fpsCap := md.AddBool("fpscap", true, "cap fps to TV specification")
	profile := md.AddString("profile", "none", "run performance check with profiling: command separated CPU, MEM, TRACE or ALL")
	log := md.AddBool("log", false, "echo debugging log to stdout")
	termType := md.AddString("term", "IMGUI", "terminal type to use in debug mode: IMGUI, COLOR, PLAIN")
	multiload := md.AddInt("multiload", -1, "force multiload byte (supercharger only; 0 to 255)")
	showFPS := md.AddBool("showfps", false, "show fps in playmode by default")

	// playmode specific arguments
	var comparisonROM *string
	var comparisonPrefs *string
	var record *bool
	var patchFile *string
	var wav *bool
	if emulationMode == emulation.ModePlay {
		comparisonROM = md.AddString("comparisonROM", "", "ROM to run in parallel for comparison")
		comparisonPrefs = md.AddString("comparisonPrefs", "", "preferences for comparison emulation")
		record = md.AddBool("record", false, "record user input to a file")
		patchFile = md.AddString("patch", "", "patch to apply to main emulation (not playback files)")
		wav = md.AddBool("wav", false, "record audio to wav file")
	}

	// debugger specific arguments
	var initScript *string
	if emulationMode == emulation.ModeDebugger {
		initScript = md.AddString("initscript", defInitScript, "script to run on debugger start")
	}

	// statsview if available
	var stats *bool
	if statsview.Available() {
		stats = md.AddBool("statsview", false, fmt.Sprintf("run stats server (%s)", statsview.Address))
	}

	// parse arguments
	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	// check remaining arguments
	if len(md.RemainingArgs()) > 1 {
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	// set debugging log echo
	if *log {
		logger.SetEcho(logger.NewColorizer(os.Stdout))
	} else {
		logger.SetEcho(nil)
	}

	if stats != nil && *stats {
		statsview.Launch(os.Stdout)
	}

	// turn off fallback ctrl-c handling. this so that the debugger can handle
	// quit events more gracefully
	//
	// we must do this before creating the emulation or we'll just end up
	// turning the emulation's interrupt handler off
	sync.state <- stateRequest{req: reqNoIntSig}

	// GUI create function
	create := func(e emulation.Emulation) (gui.GUI, terminal.Terminal, error) {
		var term terminal.Terminal
		var scr gui.GUI

		// create gui
		if *termType == "IMGUI" {
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
			scr = gui.Stub{}
		}

		// if the GUI does not supply a terminal then use a color or plain terminal
		// as a fallback
		if term == nil {
			switch strings.ToUpper(*termType) {
			default:
				fmt.Printf("! unknown terminal type (%s) defaulting to plain\n", *termType)
				fallthrough
			case "PLAIN":
				term = &plainterm.PlainTerminal{}
			case "COLOR":
				term = &colorterm.ColorTerminal{}
			}
		}

		if *showFPS {
			scr.SetFeature(gui.ReqShowFPS, true)
		}

		return scr, term, nil
	}

	// prepare new debugger instance
	dbg, err := debugger.NewDebugger(create, *spec, *fpsCap, *multiload)
	if err != nil {
		return err
	}

	// check for profiling options
	prf, err := performance.ParseProfileString(*profile)
	if err != nil {
		return err
	}

	// set up a launch function
	dbgLaunch := func() error {
		switch emulationMode {
		case emulation.ModeDebugger:
			err := dbg.StartInDebugMode(*initScript, md.GetArg(0), *mapping)
			if err != nil {
				return err
			}

		case emulation.ModePlay:
			err := dbg.StartInPlayMode(md.GetArg(0), *mapping, *record, *comparisonROM, *comparisonPrefs, *patchFile, *wav)
			if err != nil {
				return err
			}
		}

		return nil
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
		case emulation.ModeDebugger:
			s = "debugger"
		case emulation.ModePlay:
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

func disasm(md *modalflag.Modes) error {
	md.NewMode()

	mapping := md.AddString("mapping", "AUTO", "force use of cartridge mapping")
	bytecode := md.AddBool("bytecode", false, "include bytecode in disassembly")
	bank := md.AddInt("bank", -1, "show disassembly for a specific bank")

	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		attr := disassembly.ColumnAttr{
			ByteCode: *bytecode,
			Label:    true,
			Cycles:   true,
		}

		cartload, err := cartridgeloader.NewLoader(md.GetArg(0), *mapping)
		if err != nil {
			return err
		}
		defer cartload.Close()

		dsm, err := disassembly.FromCartridge(cartload)
		if err != nil {
			// print what disassembly output we do have
			if dsm != nil {
				// ignore any further errors
				_ = dsm.Write(md.Output, attr)
			}
			return err
		}

		// output entire disassembly or just a specific bank
		if *bank < 0 {
			err = dsm.Write(md.Output, attr)
		} else {
			err = dsm.WriteBank(md.Output, attr, *bank)
		}

		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	return nil
}

func perform(md *modalflag.Modes, sync *mainSync) error {
	md.NewMode()

	mapping := md.AddString("mapping", "AUTO", "force use of cartridge mapping")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL, PAL60")
	fpsCap := md.AddBool("fpscap", true, "cap FPS to specification")
	duration := md.AddString("duration", "5s", "run duration (note: there is a 2s overhead)")
	profile := md.AddString("profile", "NONE", "run performance check with profiling: command separated CPU, MEM, TRACE or ALL")
	log := md.AddBool("log", false, "echo debugging log to stdout")

	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	// set debugging log echo
	if *log {
		logger.SetEcho(logger.NewColorizer(os.Stdout))
	} else {
		logger.SetEcho(nil)
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		cartload, err := cartridgeloader.NewLoader(md.GetArg(0), *mapping)
		if err != nil {
			return err
		}
		defer cartload.Close()

		// check for profiling options
		p, err := performance.ParseProfileString(*profile)
		if err != nil {
			return err
		}

		// run performance check
		err = performance.Check(md.Output, p, cartload, *spec, *fpsCap, *duration)
		if err != nil {
			return err
		}

		// deliberately not saving gui preferences because we don't want any
		// changes to the performance window impacting the play mode

	default:
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	return nil
}

func regress(md *modalflag.Modes, sync *mainSync) error {
	md.NewMode()
	md.AddSubModes("RUN", "LIST", "DELETE", "ADD", "REDUX", "CLEANUP")

	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	switch md.Mode() {
	case "RUN":
		md.NewMode()

		// no additional arguments
		verbose := md.AddBool("verbose", false, "output more detail (eg. error messages)")

		p, err := md.Parse()
		if err != nil || p != modalflag.ParseContinue {
			return err
		}

		err = regression.RegressRun(md.Output, *verbose, md.RemainingArgs())
		if err != nil {
			return err
		}

	case "LIST":
		md.NewMode()

		// no additional arguments

		p, err := md.Parse()
		if err != nil || p != modalflag.ParseContinue {
			return err
		}

		switch len(md.RemainingArgs()) {
		case 0:
			err := regression.RegressList(md.Output)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("no additional arguments required for %s mode", md)
		}

	case "DELETE":
		md.NewMode()

		answerYes := md.AddBool("yes", false, "answer yes to confirmation")

		p, err := md.Parse()
		if err != nil || p != modalflag.ParseContinue {
			return err
		}

		switch len(md.RemainingArgs()) {
		case 0:
			return fmt.Errorf("database key required for %s mode", md)
		case 1:

			// use stdin for confirmation unless "yes" flag has been sent
			var confirmation io.Reader
			if *answerYes {
				confirmation = &yesReader{}
			} else {
				confirmation = os.Stdin
			}

			err := regression.RegressDelete(md.Output, confirmation, md.GetArg(0))
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("only one entry can be deleted at at time")
		}

	case "ADD":
		return regressAdd(md)

	case "REDUX":
		md.NewMode()

		answerYes := md.AddBool("yes", false, "always answer yes to confirmation")

		p, err := md.Parse()
		if err != nil || p != modalflag.ParseContinue {
			return err
		}

		var confirmation io.Reader
		if *answerYes {
			confirmation = &yesReader{}
		} else {
			confirmation = os.Stdin
		}

		return regression.RegressRedux(md.Output, confirmation)

	case "CLEANUP":
		md.NewMode()

		answerYes := md.AddBool("yes", false, "always answer yes to confirmation")

		p, err := md.Parse()
		if err != nil || p != modalflag.ParseContinue {
			return err
		}

		var confirmation io.Reader
		if *answerYes {
			confirmation = &yesReader{}
		} else {
			confirmation = os.Stdin
		}

		return regression.RegressCleanup(md.Output, confirmation)
	}

	return nil
}

func regressAdd(md *modalflag.Modes) error {
	md.NewMode()

	mode := md.AddString("mode", "", "type of regression entry")
	notes := md.AddString("notes", "", "additional annotation for the database")
	mapping := md.AddString("mapping", "AUTO", "force use of cartridge mapping [non-playback]")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL, PAL60 [non-playback]")
	numframes := md.AddInt("frames", 10, "number of frames to run [non-playback]")
	state := md.AddString("state", "", "record emulator state at every CPU step [non-playback]")
	log := md.AddBool("log", false, "echo debugging log to stdout")

	md.AdditionalHelp(
		`The regression test to be added can be the path to a cartridge file or a previously
recorded playback file. For playback files, the flags marked [non-playback] do not make
sense and will be ignored.

Available modes are VIDEO, PLAYBACK and LOG. If not mode is explicitly given then
VIDEO will be used for ROM files and PLAYBACK will be used for playback recordings.

Value for the -state flag can be one of TV, PORTS, TIMER, CPU and can be used
with the default VIDEO mode.

The -log flag intructs the program to echo the log to the console. Do not confuse this
with the LOG mode. Note that asking for log output will suppress regression progress meters.`)

	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	// set debugging log echo
	if *log {
		logger.SetEcho(logger.NewColorizer(os.Stdout))
		md.Output = &nopWriter{}
	} else {
		logger.SetEcho(nil)
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge or playback file required for %s mode", md)
	case 1:
		var reg regression.Regressor

		if *mode == "" {
			if err := recorder.IsPlaybackFile(md.GetArg(0)); err == nil {
				*mode = "PLAYBACK"
			} else if !curated.Is(err, recorder.NotAPlaybackFile) {
				return err
			} else {
				*mode = "VIDEO"
			}
		}

		switch strings.ToUpper(*mode) {
		case "VIDEO":
			cartload, err := cartridgeloader.NewLoader(md.GetArg(0), *mapping)
			if err != nil {
				return err
			}
			defer cartload.Close()

			statetype, err := regression.NewStateType(*state)
			if err != nil {
				return err
			}

			reg = &regression.VideoRegression{
				CartLoad:  cartload,
				TVtype:    strings.ToUpper(*spec),
				NumFrames: *numframes,
				State:     statetype,
				Notes:     *notes,
			}
		case "PLAYBACK":
			// check and warn if unneeded arguments have been specified
			md.Visit(func(flg string) {
				if flg == "frames" {
					fmt.Printf("! ignored %s flag when adding playback entry\n", flg)
				}
			})

			reg = &regression.PlaybackRegression{
				Script: md.GetArg(0),
				Notes:  *notes,
			}
		case "LOG":
			cartload, err := cartridgeloader.NewLoader(md.GetArg(0), *mapping)
			if err != nil {
				return err
			}
			defer cartload.Close()

			reg = &regression.LogRegression{
				CartLoad:  cartload,
				TVtype:    strings.ToUpper(*spec),
				NumFrames: *numframes,
				Notes:     *notes,
			}
		}

		err := regression.RegressAdd(md.Output, reg)
		if err != nil {
			// using carriage return (without newline) at beginning of error
			// message because we want to overwrite the last output from
			// RegressAdd()
			return fmt.Errorf("\rerror adding regression test: %v", err)
		}
	default:
		return fmt.Errorf("regression tests can only be added one at a time")
	}

	return nil
}

// nopWriter is an empty writer.
type nopWriter struct{}

func (*nopWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

// yesReader always returns 'y' when it is read.
type yesReader struct{}

func (*yesReader) Read(p []byte) (n int, err error) {
	p[0] = 'y'
	return 1, nil
}
