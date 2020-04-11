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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/colorterm"
	"github.com/jetsetilly/gopher2600/debugger/terminal/plainterm"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/gui/sdldebug"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui"
	"github.com/jetsetilly/gopher2600/gui/sdlimgui_play"
	"github.com/jetsetilly/gopher2600/gui/sdlplay"
	"github.com/jetsetilly/gopher2600/modalflag"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/performance"
	"github.com/jetsetilly/gopher2600/playmode"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/regression"
	"github.com/jetsetilly/gopher2600/television"
	"github.com/jetsetilly/gopher2600/wavwriter"
)

const defaultInitScript = "debuggerInit"

type stateReq int

const (
	// main thread should end as soon as possible
	reqQuit stateReq = iota

	// reset interrupt signal handling. used when an alternative
	// handler is more appropriate. for example, the playMode and Debugger
	// package provide a mode specific handler.
	reqNoIntSig
)

// GuiCreator facilitates the creation, servicing and desctruction of GUIs
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
// handling (including creation) to occur on the main thread
type mainSync struct {
	state   chan stateReq
	creator chan func() (GuiCreator, error)

	// the result of creator will be returned on either of these two channels.
	creation      chan GuiCreator
	creationError chan error
}

// #mainthread
func main() {
	sync := &mainSync{
		state:         make(chan stateReq),
		creator:       make(chan func() (GuiCreator, error)),
		creation:      make(chan GuiCreator),
		creationError: make(chan error),
	}

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
			} else {
				sync.creation <- gui
			}

		case state := <-sync.state:
			switch state {
			case reqQuit:
				done = true
			case reqNoIntSig:
				signal.Reset(os.Interrupt)
			}

		default:
			// if an instance of gui.Events has been sent to us via sync.events
			// then call Service()
			if gui != nil {
				gui.Service()
			}
		}
	}

	// destroy gui
	if gui != nil {
		gui.Destroy(os.Stderr)
	}

	fmt.Print("\r")
}

// launch is called from main() as a goroutine. uses mainSync instance to
// indicate gui creation and to quit
func launch(sync *mainSync) {
	defer func() {
		sync.state <- reqQuit
	}()

	// we generate random numbers in some places. seed the generator with the
	// current time
	// rand.Seed(int64(time.Now().Second()))

	md := &modalflag.Modes{Output: os.Stdout}
	md.NewArgs(os.Args[1:])
	md.NewMode()
	md.AddSubModes("RUN", "PLAY", "DEBUG", "DISASM", "PERFORMANCE", "REGRESS")

	p, err := md.Parse()
	switch p {
	case modalflag.ParseHelp:
		os.Exit(0)
	case modalflag.ParseError:
		fmt.Printf("* %s\n", err)
		os.Exit(10)
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
		err = regress(md)
	}

	if err != nil {
		fmt.Printf("* %s\n", err)
		os.Exit(20)
	}
}

func play(md *modalflag.Modes, sync *mainSync) error {
	md.NewMode()

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
	scaling := md.AddFloat64("scale", 3.0, "television scaling")
	stable := md.AddBool("stable", true, "wait for stable frame before opening display")
	pixelPerfect := md.AddBool("pixelperfect", false, "pixel perfect display")
	fpsCap := md.AddBool("fpscap", true, "cap fps to specification")
	record := md.AddBool("record", false, "record user input to a file")
	wav := md.AddString("wav", "", "record audio to wav file")
	patchFile := md.AddString("patch", "", "patch file to apply (cartridge args only)")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		cartload := cartridgeloader.Loader{
			Filename: md.GetArg(0),
			Format:   *cartFormat,
		}

		tv, err := television.NewTelevision(*spec)
		if err != nil {
			return errors.New(errors.PlayError, err)
		}
		defer tv.End()

		// set fps cap
		tv.SetFPSCap(*fpsCap)

		// add wavwriter mixer if wav argument has been specified
		if *wav != "" {
			aw, err := wavwriter.New(*wav)
			if err != nil {
				return errors.New(errors.PlayError, err)
			}
			tv.AddAudioMixer(aw)
		}

		// create gui
		if *pixelPerfect {
			sync.creator <- func() (GuiCreator, error) {
				return sdlplay.NewSdlPlay(tv, float32(*scaling))
			}
		} else {
			sync.creator <- func() (GuiCreator, error) {
				return sdlimgui_play.NewSdlImguiPlay(tv)
			}
		}

		// wait for creator result
		var scr gui.GUI
		select {
		case g := <-sync.creation:
			scr = g.(gui.GUI)
		case err := <-sync.creationError:
			return errors.New(errors.PlayError, err)
		}

		// turn off fallback ctrl-c handling. this so that the playmode can
		// end playback recordings gracefully
		sync.state <- reqNoIntSig

		// set scaling value
		err = scr.SetFeature(gui.ReqSetScale, float32(*scaling))
		if err != nil {
			return err
		}

		err = playmode.Play(tv, scr, *stable, *record, cartload, *patchFile)
		if err != nil {
			return err
		}

		if *record {
			fmt.Println("! recording completed")
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
		return errors.New(errors.DebuggerError, err)
	}

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
	termType := md.AddString("term", "IMGUI", "terminal type to use in debug mode: IMGUI, COLOR, PLAIN")
	initScript := md.AddString("initscript", defInitScript, "script to run on debugger start")
	profile := md.AddBool("profile", false, "run debugger through cpu profiler")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	tv, err := television.NewTelevision(*spec)
	if err != nil {
		return errors.New(errors.DebuggerError, err)
	}
	defer tv.End()

	var term terminal.Terminal

	// decide which gui to use
	if *termType == "IMGUI" {
		sync.creator <- func() (GuiCreator, error) {
			return sdlimgui.NewSdlImgui(tv)
		}
	} else {

		// notify main thread of new gui creator
		sync.creator <- func() (GuiCreator, error) {
			return sdldebug.NewSdlDebug(tv, 2.0)
		}
	}

	// wait for creator result
	var scr gui.GUI
	select {
	case g := <-sync.creation:
		scr = g.(gui.GUI)
	case err := <-sync.creationError:
		return errors.New(errors.PlayError, err)
	}

	// if gui implements the terminal.Broker interface use that terminal
	// as a preference
	if b, ok := scr.(terminal.Broker); ok {
		term = b.GetTerminal()
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
	sync.state <- reqNoIntSig

	// prepare new debugger instance
	dbg, err := debugger.NewDebugger(tv, scr, term)
	if err != nil {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)

	case 1:
		// set up a running function
		dbgRun := func() error {
			cartload := cartridgeloader.Loader{
				Filename: md.GetArg(0),
				Format:   *cartFormat,
			}
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

	return nil
}

func disasm(md *modalflag.Modes) error {
	md.NewMode()

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")
	bytecode := md.AddBool("bytecode", false, "include bytecode in disassembly")
	raw := md.AddBool("raw", false, "raw disassembly. show every byte with the disasm decision.")
	bank := md.AddInt("bank", -1, "show disassembly for a specific bank")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		attr := disassembly.WriteAttr{
			ByteCode: *bytecode,
			Raw:      *raw,
		}

		cartload := cartridgeloader.Loader{
			Filename: md.GetArg(0),
			Format:   *cartFormat,
		}
		dsm, err := disassembly.FromCartridge(cartload)
		if err != nil {
			// print what disassembly output we do have
			if dsm != nil {
				// ignore any further errors
				_ = dsm.Write(md.Output, attr)
			}
			return errors.New(errors.DisassemblyError, err)
		}

		// output entire disassembly or just a specific bank
		if *bank < 0 {
			err = dsm.Write(md.Output, attr)
		} else {
			err = dsm.WriteBank(md.Output, attr, *bank)
		}

		if err != nil {
			return errors.New(errors.DisassemblyError, err)
		}
	default:
		return fmt.Errorf("too many arguments for %s mode", md)
	}

	return nil
}

func perform(md *modalflag.Modes, sync *mainSync) error {
	md.NewMode()

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
	display := md.AddBool("display", false, "display TV output")
	scaling := md.AddFloat64("scale", 3.0, "display scaling (only valid if -display=true")
	pixelPerfect := md.AddBool("pixelperfect", false, "pixel perfect display")
	fpsCap := md.AddBool("fpscap", true, "cap FPS to specification (only valid if -display=true)")
	duration := md.AddString("duration", "5s", "run duration (note: there is a 2s overhead)")
	profile := md.AddBool("profile", false, "produce cpu and memory profiling reports")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		cartload := cartridgeloader.Loader{
			Filename: md.GetArg(0),
			Format:   *cartFormat,
		}

		tv, err := television.NewTelevision(*spec)
		if err != nil {
			return errors.New(errors.PerformanceError, err)
		}
		defer tv.End()

		tv.SetFPSCap(*fpsCap)

		if *display {
			// create gui
			if *pixelPerfect {
				sync.creator <- func() (GuiCreator, error) {
					return sdlplay.NewSdlPlay(tv, float32(*scaling))
				}
			} else {
				sync.creator <- func() (GuiCreator, error) {
					return sdlimgui_play.NewSdlImguiPlay(tv)
				}
			}

			// wait for creator result
			var scr gui.GUI
			select {
			case g := <-sync.creation:
				scr = g.(gui.GUI)
			case err := <-sync.creationError:
				return errors.New(errors.PlayError, err)
			}

			// set scaling value
			err = scr.SetFeature(gui.ReqSetScale, float32(*scaling))
			if err != nil {
				return err
			}

			// show gui
			err = scr.SetFeature(gui.ReqSetVisibility, true)
			if err != nil {
				return errors.New(errors.PerformanceError, err)
			}
		}

		err = performance.Check(md.Output, *profile, tv, *duration, cartload)
		if err != nil {
			return err
		}

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

func regress(md *modalflag.Modes) error {
	md.NewMode()
	md.AddSubModes("RUN", "LIST", "DELETE", "ADD")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch md.Mode() {
	case "RUN":
		md.NewMode()

		// no additional arguments
		verbose := md.AddBool("verbose", false, "output more detail (eg. error messages)")
		failOnError := md.AddBool("fail", false, "fail on error")

		p, err := md.Parse()
		if p != modalflag.ParseContinue {
			return err
		}

		err = regression.RegressRunTests(md.Output, *verbose, *failOnError, md.RemainingArgs())
		if err != nil {
			return err
		}

	case "LIST":
		md.NewMode()

		// no additional arguments

		p, err := md.Parse()
		if p != modalflag.ParseContinue {
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
		if p != modalflag.ParseContinue {
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
			return fmt.Errorf("only one entry can be deleted at at time when using %s mode", md)
		}

	case "ADD":
		return regressAdd(md)
	}

	return nil
}

func regressAdd(md *modalflag.Modes) error {
	md.NewMode()

	cartFormat := md.AddString("cartformat", "AUTO", "force use of cartridge format")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL [cartridge args only]")
	numframes := md.AddInt("frames", 10, "number of frames to run [cartridge args only]")
	state := md.AddBool("state", false, "record TV state at every CPU step [cartrdige args only]")
	mode := md.AddString("mode", "video", "type of digest to create [cartridge args only]")
	notes := md.AddString("notes", "", "annotation for the database")

	md.AdditionalHelp("The regression test to be added can be the path to a cartrige file or a previously recorded playback file. For playback files, the flags marked [cartridge args only] do not make sense and will be ignored.")

	p, err := md.Parse()
	if p != modalflag.ParseContinue {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge or playback file required for %s mode", md)
	case 1:
		var rec regression.Regressor

		if recorder.IsPlaybackFile(md.GetArg(0)) {
			// check and warn if unneeded arguments have been specified
			md.Visit(func(flg string) {
				if flg == "frames" {
					fmt.Printf("! ignored %s flag when adding playback entry\n", flg)
				}
			})

			rec = &regression.PlaybackRegression{
				Script: md.GetArg(0),
				Notes:  *notes,
			}
		} else {
			cartload := cartridgeloader.Loader{
				Filename: md.GetArg(0),
				Format:   *cartFormat,
			}

			// parse digest mode, failing if string is not recognised
			m, err := regression.ParseDigestMode(*mode)
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			rec = &regression.DigestRegression{
				Mode:      m,
				CartLoad:  cartload,
				TVtype:    strings.ToUpper(*spec),
				NumFrames: *numframes,
				State:     *state,
				Notes:     *notes,
			}
		}

		err := regression.RegressAdd(md.Output, rec)
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
