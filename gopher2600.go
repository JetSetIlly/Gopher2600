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
	"gopher2600/cartridgeloader"
	"gopher2600/debugger"
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/colorterm"
	"gopher2600/debugger/terminal/plainterm"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/gui/sdldebug"
	"gopher2600/gui/sdlplay"
	"gopher2600/modalflag"
	"gopher2600/paths"
	"gopher2600/performance"
	"gopher2600/playmode"
	"gopher2600/recorder"
	"gopher2600/regression"
	"gopher2600/television"
	"gopher2600/wavwriter"
	"io"
	"os"
	"strings"
)

const defaultInitScript = "debuggerInit"

// communication between the main() function and the launch() function. this is
// required because many gui solutions (notably SDL) require window event
// handling (including creation) to occur on the main thread
type mainSync struct {
	quit    chan bool
	creator chan func() (gui.GUI, error)

	// the result of creator will be returned on either of these two channels.
	creation      chan gui.GUI
	creationError chan error
}

// #main #mainthread

func main() {
	sync := &mainSync{
		quit:          make(chan bool),
		creator:       make(chan func() (gui.GUI, error)),
		creation:      make(chan gui.GUI),
		creationError: make(chan error),
	}

	// launch program as a go routine. further communication is  through
	// the mainSync instance
	go launch(sync)

	// loop until quit is true. every iteration of the loop we listen for:
	//
	//  1. quit signals
	//  2. new gui creation functions
	//  3. anything in the Service() function of the most recently created GUI
	//
	quit := false
	var gui gui.GUI
	for !quit {
		select {
		case creator := <-sync.creator:
			var err error
			gui, err = creator()
			if err != nil {
				sync.creationError <- err
			} else {
				sync.creation <- gui
			}

		case quit = <-sync.quit:

		default:
			// if an instance of gui.Events has been sent to us via sync.events
			// then call Service()
			if gui != nil {
				gui.Service()
			}
		}
	}
}

// launch is called from main() as a goroutine. uses mainSync instance to
// indicate gui creation and quit events
func launch(sync *mainSync) {
	defer func() {
		sync.quit <- true
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
	fpscap := md.AddBool("fpscap", true, "cap fps to specification")
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

		// add wavwriter mixer if wav argument has been specified
		if *wav != "" {
			aw, err := wavwriter.New(*wav)
			if err != nil {
				return errors.New(errors.PlayError, err)
			}
			tv.AddAudioMixer(aw)
		}

		// notify main thread of new gui creator
		sync.creator <- func() (gui.GUI, error) {
			return sdlplay.NewSdlPlay(tv, float32(*scaling))
		}

		// wait for creator result
		var scr gui.GUI
		select {
		case scr = <-sync.creation:
		case err := <-sync.creationError:
			return errors.New(errors.PlayError, err)
		}

		err = playmode.Play(tv, scr, *stable, *fpscap, *record, cartload, *patchFile)
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
	termType := md.AddString("term", "COLOR", "terminal type to use in debug mode: COLOR, PLAIN")
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

	// notify main thread of new gui creator
	sync.creator <- func() (gui.GUI, error) {
		return sdldebug.NewSdlDebug(tv, 2.0)
	}

	// wait for creator result
	var scr gui.GUI
	select {
	case scr = <-sync.creation:
	case err := <-sync.creationError:
		return errors.New(errors.PlayError, err)
	}

	// start debugger with choice of interface and cartridge
	var cons terminal.Terminal

	switch strings.ToUpper(*termType) {
	default:
		fmt.Printf("! unknown terminal type (%s) defaulting to plain\n", *termType)
		fallthrough
	case "PLAIN":
		cons = &plainterm.PlainTerminal{}
	case "COLOR":
		cons = &colorterm.ColorTerminal{}
	}

	dbg, err := debugger.NewDebugger(tv, scr, cons)
	if err != nil {
		return err
	}

	switch len(md.RemainingArgs()) {
	case 0:
		return fmt.Errorf("2600 cartridge required for %s mode", md)
	case 1:
		runner := func() error {
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

		if *profile {
			err := performance.ProfileCPU("debug.cpu.profile", runner)
			if err != nil {
				return err
			}
			err = performance.ProfileMem("debug.mem.profile")
			if err != nil {
				return err
			}
		} else {
			err := runner()
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
		dsm, err := disassembly.FromCartridge(cartload)
		if err != nil {
			// print what disassembly output we do have
			if dsm != nil {
				// ignore any further errors
				_ = dsm.Write(md.Output, *bytecode)
			}
			return errors.New(errors.DisassemblyError, err)
		}
		err = dsm.Write(md.Output, *bytecode)
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
	display := md.AddBool("display", false, "display TV output")
	fpscap := md.AddBool("fpscap", true, "cap FPS to specification (only valid if -display=true)")
	scaling := md.AddFloat64("scale", 3.0, "display scaling (only valid if -display=true")
	spec := md.AddString("tv", "AUTO", "television specification: NTSC, PAL")
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

		if *display {
			// notify main thread of new gui creator
			sync.creator <- func() (gui.GUI, error) {
				return sdlplay.NewSdlPlay(tv, float32(*scaling))
			}

			// wait for creator result
			var scr gui.GUI
			select {
			case scr = <-sync.creation:
			case err := <-sync.creationError:
				return errors.New(errors.PlayError, err)
			}

			err = scr.SetFeature(gui.ReqSetVisibility, true)
			if err != nil {
				return errors.New(errors.PerformanceError, err)
			}

			err = scr.SetFeature(gui.ReqSetFPSCap, *fpscap)
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
		return errors.New(errors.RegressionError, err)
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
