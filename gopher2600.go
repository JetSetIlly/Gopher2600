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
	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/colorterm"
	"github.com/jetsetilly/gopher2600/debugger/terminal/plainterm"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hiscore"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/modalflag"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/performance"
	"github.com/jetsetilly/gopher2600/playmode"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/regression"
	"github.com/jetsetilly/gopher2600/wavwriter"
)

const defaultInitScript = "debuggerInit"

type stateReq = string

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
)

type stateRequest struct {
	req  stateReq
	args interface{}
}

// GuiCreator facilitates the creation, servicing and destruction of GUIs
// that need to be run in the main thread.
//
// Note that there is no Create() function because we need the freedom to
// create the GUI how we want. Instead the creator is a channel which accepts
// a function that returns an instance of GuiCreator.
type GuiCreator interface {
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

// communication between the main() function and the launch() function. this is
// required because many gui solutions (notably SDL) require window event
// handling (including creation) to occur on the main thread.
type mainSync struct {
	state   chan stateRequest
	creator chan func() (GuiCreator, error)

	// the result of creator will be returned on either of these two channels.
	creation      chan GuiCreator
	creationError chan error
}

// #mainthread
func main() {
	sync := &mainSync{
		state:         make(chan stateRequest),
		creator:       make(chan func() (GuiCreator, error)),
		creation:      make(chan GuiCreator),
		creationError: make(chan error),
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

	// loop until done is true. every iteration of the loop we listen for:
	//
	//  1. interrupt signals
	//  2. new gui creation functions
	//  3. state requests
	//  3. anything in the Service() function of the most recently created GUI
	//
	done := false
	var gui GuiCreator
	for !done {
		select {
		case <-intChan:
			fmt.Println("\r")
			done = true

		case creator := <-sync.creator:
			var err error

			// destroy existing gui
			if gui != nil {
				gui.Destroy(os.Stderr)
			}

			gui, err = creator()
			if err != nil {
				sync.creationError <- err

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
				sync.creation <- gui
			}

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
			}

		default:
			// if an instance of gui.Events has been sent to us via sync.events
			// then call Service()
			if gui != nil {
				gui.Service()
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
	md.AddSubModes("RUN", "PLAY", "DEBUG", "DISASM", "PERFORMANCE", "REGRESS", "HISCORE")

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
		err = play(md, sync)

	case "DEBUG":
		err = debug(md, sync)

	case "DISASM":
		err = disasm(md)

	case "PERFORMANCE":
		err = perform(md, sync)

	case "REGRESS":
		err = regress(md, sync)

	case "HISCORE":
		err = hiscoreServer(md)
	}

	if err != nil {
		fmt.Printf("* error in %s mode: %s\n", md.String(), err)
		sync.state <- stateRequest{req: reqQuit, args: 20}
		return
	}

	sync.state <- stateRequest{req: reqQuit}
}

func play(md *modalflag.Modes, sync *mainSync) error {
	md.NewMode()

	mapping := md.AddString("mapping", "AUTO", "force use of cartridge mapping")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
	scaling := md.AddFloat64("scale", 0.0, "television scaling")
	crt := md.AddBool("crt", true, "apply CRT post-processing")
	fpsCap := md.AddBool("fpscap", true, "cap fps to specification")
	record := md.AddBool("record", false, "record user input to a file")
	wav := md.AddString("wav", "", "record audio to wav file")
	patchFile := md.AddString("patch", "", "patch file to apply (cartridge args only)")
	hiscore := md.AddBool("hiscore", false, "contact hiscore server [EXPERIMENTAL]")
	log := md.AddBool("log", false, "echo debugging log to stdout")
	useSavekey := md.AddBool("savekey", false, "use savekey in player 1 port")

	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	// set debugging log echo
	if *log {
		logger.SetEcho(os.Stdout)
	} else {
		logger.SetEcho(nil)
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		cartload := cartridgeloader.NewLoader(md.GetArg(0), *mapping)

		tv, err := television.NewTelevision(*spec)
		if err != nil {
			return err
		}
		defer tv.End()

		// set fps cap
		tv.SetFPSCap(*fpsCap)

		// add wavwriter mixer if wav argument has been specified
		if *wav != "" {
			aw, err := wavwriter.New(*wav)
			if err != nil {
				return err
			}
			tv.AddAudioMixer(aw)
		}

		// create gui
		sync.creator <- func() (GuiCreator, error) {
			return sdlimgui.NewSdlImgui(tv, true)
		}

		// wait for creator result
		var scr gui.GUI
		select {
		case g := <-sync.creation:
			scr = g.(gui.GUI)

			err = scr.ReqFeature(gui.ReqSetPlaymode, true)
			if err != nil {
				return err
			}

			if *crt {
				err = scr.ReqFeature(gui.ReqCRTeffects, true)
				if err != nil {
					return err
				}
			}

		case err := <-sync.creationError:
			return err
		}

		// turn off fallback ctrl-c handling. this so that the playmode can
		// end playback recordings gracefully
		sync.state <- stateRequest{req: reqNoIntSig}

		// set scaling value
		if *scaling > 0.0 {
			err = scr.ReqFeature(gui.ReqSetScale, float32(*scaling))
			if err != nil {
				return err
			}
		}

		err = playmode.Play(tv, scr, *record, cartload, *patchFile, *hiscore, *useSavekey)
		if err != nil {
			return err
		}

		if *record {
			fmt.Println("! recording completed")
		}

		// save preferences before finishing successfully
		err = scr.ReqFeature(gui.ReqSavePrefs)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	return nil
}

func debug(md *modalflag.Modes, sync *mainSync) error {
	md.NewMode()

	defInitScript, err := paths.ResourcePath("", defaultInitScript)
	if err != nil {
		return err
	}

	mapping := md.AddString("mapping", "AUTO", "force use of cartridge mapping")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
	termType := md.AddString("term", "IMGUI", "terminal type to use in debug mode: IMGUI, COLOR, PLAIN")
	initScript := md.AddString("initscript", defInitScript, "script to run on debugger start")
	profile := md.AddBool("profile", false, "run debugger through cpu profiler")
	useSavekey := md.AddBool("savekey", false, "use savekey in player 1 port")

	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	tv, err := television.NewTelevision(*spec)
	if err != nil {
		return err
	}
	defer tv.End()

	var term terminal.Terminal
	var scr gui.GUI

	// create gui
	if *termType == "IMGUI" {
		sync.creator <- func() (GuiCreator, error) {
			return sdlimgui.NewSdlImgui(tv, false)
		}

		// wait for creator result
		select {
		case g := <-sync.creation:
			scr = g.(gui.GUI)
		case err := <-sync.creationError:
			return err
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

	// turn off fallback ctrl-c handling. this so that the debugger can handle
	// quit events with a confirmation request. it also allows the debugger to
	// use ctrl-c events to interrupt execution of the emulation without
	// quitting the debugger itself
	sync.state <- stateRequest{req: reqNoIntSig}

	// prepare new debugger instance
	dbg, err := debugger.NewDebugger(tv, scr, term, *useSavekey)
	if err != nil {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)

	case 1:
		// set up a running function
		dbgRun := func() error {
			cartload := cartridgeloader.NewLoader(md.GetArg(0), *mapping)

			err := dbg.Start(*initScript, cartload)
			if err != nil {
				return err
			}
			return nil
		}

		// if profile generation has been requested then pass the dbgRun()
		// function prepared above, through the ProfileCPU() command
		if *profile {
			err := performance.ProfileCPU("debug.cpu.profile", dbgRun)
			if err != nil {
				return err
			}
			err = performance.ProfileMem("debug.mem.profile")
			if err != nil {
				return err
			}
		} else {
			// no profile required so run dbgRun() function as normal
			err := dbgRun()
			if err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	// save preferences before finishing successfully
	err = scr.ReqFeature(gui.ReqSavePrefs)
	if err != nil {
		return err
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
		attr := disassembly.WriteAttr{
			ByteCode: *bytecode,
		}

		cartload := cartridgeloader.NewLoader(md.GetArg(0), *mapping)

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
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
	display := md.AddBool("display", false, "display TV output")
	scaling := md.AddFloat64("scale", 0.0, "display scaling (only valid if -display=true")
	fpsCap := md.AddBool("fpscap", true, "cap FPS to specification (only valid if -display=true)")
	duration := md.AddString("duration", "5s", "run duration (note: there is a 2s overhead)")
	profile := md.AddBool("profile", false, "produce cpu and memory profiling reports")

	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		cartload := cartridgeloader.NewLoader(md.GetArg(0), *mapping)

		tv, err := television.NewTelevision(*spec)
		if err != nil {
			return err
		}
		defer tv.End()

		tv.SetFPSCap(*fpsCap)

		if *display {
			// create gui
			sync.creator <- func() (GuiCreator, error) {
				return sdlimgui.NewSdlImgui(tv, true)
			}

			// wait for creator result
			var scr gui.GUI
			select {
			case g := <-sync.creation:
				scr = g.(gui.GUI)
			case err := <-sync.creationError:
				return err
			}

			// set scaling value
			if *scaling > 0.0 {
				err = scr.ReqFeature(gui.ReqSetScale, float32(*scaling))
				if err != nil {
					return err
				}
			}
		}

		err = performance.Check(md.Output, *profile, tv, *duration, cartload)
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

type yesReader struct{}

func (*yesReader) Read(p []byte) (n int, err error) {
	p[0] = 'y'
	return 1, nil
}

func regress(md *modalflag.Modes, sync *mainSync) error {
	md.NewMode()
	md.AddSubModes("RUN", "LIST", "DELETE", "ADD")

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

		// turn off default sigint handling
		sync.state <- stateRequest{req: reqNoIntSig}

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
	}

	return nil
}

func regressAdd(md *modalflag.Modes) error {
	md.NewMode()

	mode := md.AddString("mode", "", "type of regression entry")
	notes := md.AddString("notes", "", "additional annotation for the database")
	mapping := md.AddString("mapping", "AUTO", "force use of cartridge mapping [non-playback]")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL [non-playback]")
	numframes := md.AddInt("frames", 10, "number of frames to run [non-playback]")
	state := md.AddString("state", "", "record emulator state at every CPU step [non-playback]")
	log := md.AddBool("log", false, "echo debugging log to stdout")

	md.AdditionalHelp(
		`The regression test to be added can be the path to a cartridge file or a previously
recorded playback file. For playback files, the flags marked [non-playback] do not make
sense and will be ignored.

Available modes are VIDEO, PLAYBACK and LOG. If not mode is explicitly given then
VIDEO will be used for ROM files and PLAYBACK will be used for playback recordings.

The -log flag intructs the program to echo the log to the console. Do not confuse this
with the LOG mode. Note that asking for log output will suppress regression progress meters.`)

	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	// set debugging log echo
	if *log {
		logger.SetEcho(os.Stdout)
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
			if recorder.IsPlaybackFile(md.GetArg(0)) {
				*mode = "PLAYBACK"
			} else {
				*mode = "VIDEO"
			}
		}

		switch strings.ToUpper(*mode) {
		case "VIDEO":
			cartload := cartridgeloader.NewLoader(md.GetArg(0), *mapping)

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
			cartload := cartridgeloader.NewLoader(md.GetArg(0), *mapping)

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

func hiscoreServer(md *modalflag.Modes) error {
	md.NewMode()
	md.AddSubModes("ABOUT", "SETSERVER", "LOGIN", "LOGOFF")
	md.AdditionalHelp("Hiscore server support is EXPERIMENTAL")

	p, err := md.Parse()
	if err != nil || p != modalflag.ParseContinue {
		return err
	}

	switch md.Mode() {
	case "ABOUT":
		fmt.Println("The hiscore server is experimental and is not currently fully functioning")

	case "LOGIN":
		md.NewMode()
		p, err := md.Parse()
		if err != nil || p != modalflag.ParseContinue {
			return err
		}

		username := ""
		args := md.RemainingArgs()

		switch len(args) {
		case 0:
			// an empty string is okay
		case 1:
			username = args[0]
		default:
			return fmt.Errorf("too many arguments for %s", md)
		}

		err = hiscore.Login(os.Stdin, os.Stdout, username)
		if err != nil {
			return err
		}

	case "LOGOFF":
		err = hiscore.Logoff()
		if err != nil {
			return err
		}

	case "SETSERVER":
		md.NewMode()
		p, err := md.Parse()
		if err != nil || p != modalflag.ParseContinue {
			return err
		}

		server := ""
		args := md.RemainingArgs()

		switch len(args) {
		case 0:
			// an empty string is okay
		case 1:
			server = args[0]
		default:
			return fmt.Errorf("too many arguments for %s", md)
		}

		err = hiscore.SetServer(os.Stdin, os.Stdout, server)
		if err != nil {
			return err
		}
	}

	return nil
}

// nopWriter is an empty writer.
type nopWriter struct{}

func (*nopWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}
