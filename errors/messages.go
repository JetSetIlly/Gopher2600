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

package errors

// error messages
const (
	// panics
	PanicError = "panic: %v: %v"

	// sentinals
	UserInterrupt = "user interrupt"
	UserQuit      = "user quit"
	ScriptEnd     = "end of script (%v)"
	PowerOff      = "emulated machine has been powered off"
	TVOutOfSpec   = "tv out of spec: %v"

	// program modes
	PlayError        = "error emulating vcs: %v"
	DebuggerError    = "error debugging vcs: %v"
	PerformanceError = "error during performance profiling: %v"
	DisassemblyError = "error during disassembly: %v"

	// debugger
	InvalidTarget   = "invalid target (%v)"
	CommandError    = "%v"
	TerminalError   = "%v"
	GUIEventError   = "%v"
	BreakpointError = "breakpoint error: %v"

	// commandline
	ParserError     = "parser error: %v"
	HelpError       = "help error: %v"
	ValidationError = "%v"

	// dissassembly
	DisasmError    = "disasm error: %v"
	IterationError = "disasm iteration error: %v"

	// script
	ScriptFileError       = "script error: %v"
	ScriptFileUnavailable = "script error: cannot open script file (%v)"
	ScriptRunError        = "script error: use of '%v' is not allowed in scripts [%v::%d]"
	ScriptScribeError     = "script scribe error: %v"

	// recorder
	RecordingError    = "recording error: %v"
	PlaybackError     = "playback error: %v"
	PlaybackHashError = "playback error: hash error: %v"

	// database
	DatabaseError           = "database error: %v"
	DatabaseReadError       = "database error: %v [line %d]"
	DatabaseSelectEmpty     = "database error: no selected entries"
	DatabaseKeyError        = "database error: no such key in database [%v]"
	DatabaseFileUnavailable = "database error: cannot open database (%v)"

	// regression
	RegressionError         = "regression: %v"
	RegressionDigestError   = "digest regression: %v"
	RegressionPlaybackError = "playback regression: %v"

	// setup
	SetupError           = "setup error: %v"
	SetupPanelError      = "panel setup: %v"
	SetupPatchError      = "patch setup: %v"
	SetupTelevisionError = "tv setup: %v"

	// patch
	PatchError = "patch error: %v"

	// symbols
	SymbolsFileError       = "symbols error: error processing symbols file: %v"
	SymbolsFileUnavailable = "symbols error: no symbols file for %v"
	SymbolUnknown          = "symbols error: unrecognised symbol (%v)"

	// cartridgeloader
	CartridgeLoader = "cartridge loading error: %v"

	// vcs
	PolycounterError = "polycounter error: %v"

	// cpu
	InvalidResult          = "cpu error: %v"
	InvalidDuringExecution = "cpu error: invalid operation mid-instruction (%v)"
	CPUBug                 = "cpu bug: %v"
	// TODO: remove this once all opcodes are defined/implemented
	UnimplementedInstruction = "cpu error: unimplemented instruction (%#02x) at (%#04x)"

	// memory
	UnpokeableAddress = "memory error: cannot poke address (%v)"
	UnpeekableAddress = "memory error: cannot peek address (%v)"
	MemoryBusError    = "memory error: inaccessible address (%v)"

	// cartridges
	CartridgeError        = "cartridge error: %v"
	CartridgeEjected      = "cartridge error: no cartridge attached"
	CartridgeNotMappable  = "cartridge error: bank %d can not be mapped to that address (%#04x)"
	CartridgePatchOOB     = "cartrdige error: patch offset too high (%#04x)"
	CartridgeStaticArea   = "cartridge error: static area: %v"
	SuperchargerError     = "cartridge error: AR: %v"
	SuperchargerEndOfTape = "end of tape"

	// input
	InputError     = "input error: %v"
	UnhandledEvent = "unhandled event: %s does not support event %v"
	KeyboardError  = "keyboard: %v"
	StickError     = "stick: %v"
	PaddleError    = "paddle: %v"

	// television
	UnknownTVRequest = "television error: unsupported request (%v)"
	Television       = "television error: %v"

	// digests
	VideoDigest = "video digest: %v"
	AudioDigest = "audio digest: %v"

	// audio2wav
	WavWriter = "wav writer: %v"

	// gui
	UnsupportedGUIRequest = "unsupported request (%v)"
	SDLDebug              = "sdldebug: %v"
	SDLPlay               = "sdlplay: %v"
	SDLImgui              = "sdlimgui: %v"

	// hiscore server
	HiScore = "hiscore: %v"

	// linter
	Linter = "linter: %v"

	// prefs
	Prefs         = "prefs: %v"
	PrefsNoFile   = "prefs: no file (%s)"
	PrefsNotValid = "prefs: not a valid prefs file (%s)"
)
