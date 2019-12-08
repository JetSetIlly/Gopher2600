package errors

var messages = map[Errno]string{
	// panics
	PanicError: "panic: %v: %v",

	// sentinals
	UserInterrupt:   "user interrupt",
	UserSuspend:     "user suspend",
	ScriptEnd:       "end of script (%v)",
	PowerOff:        "emulated machine has been powered off",
	PeriphUnplugged: "controller unplugged from %v",
	TVOutOfSpec:     "tv out of spec: %v",

	// program modes
	PlayError:        "error emulating vcs: %v",
	DebuggerError:    "error debugging vcs: %v",
	PerformanceError: "error during performance profiling: %v",
	DisasmError:      "error during disassembly: %v",

	// debugger
	ParserError:          "parser error: %v: %v (char %d)", // first placeholder is the command definition
	ValidationError:      "%v for %v",
	InvalidTarget:        "invalid target (%v)",
	CommandError:         "%v",
	TerminalError:        "%v",
	GUIEventError:        "%v",
	ReflectionNotRunning: "reflection process is not running",

	// script
	ScriptFileError:       "script error: %v",
	ScriptFileUnavailable: "script error: cannot open script file (%v)",
	ScriptRunError:        "script error: use of '%v' is not allowed in scripts [%v::%d]",
	ScriptScribeError:     "script scribe error: %v",

	// recorder
	RecordingError:    "controller recording error: %v",
	PlaybackError:     "controller playback error: %v",
	PlaybackHashError: "controller playback error: hash error: %v",

	// database
	DatabaseError:           "database error: %v",
	DatabaseSelectEmpty:     "database error: no selected entries",
	DatabaseKeyError:        "database error: no such key in database [%v]",
	DatabaseFileUnavailable: "database error: cannot open database (%v)",

	// regression
	RegressionError:         "regression test error: %v",
	RegressionFrameError:    "regression test error: frame entry: %v",
	RegressionPlaybackError: "regression test error: playback entry: %v",

	// setup
	SetupError:      "setup error: %v",
	SetupPanelError: "setup error: panel entry: %v",

	// symbols
	SymbolsFileError:       "symbols error: error processing symbols file: %v",
	SymbolsFileUnavailable: "symbols error: no symbols file for %v",
	SymbolUnknown:          "symbols error: unrecognised symbol (%v)",

	// cartridgeloader
	CartridgeLoader: "cartridge loading error: %v",

	// vcs
	VCSError:         "vcs error: %v",
	PolycounterError: "polycounter error: %v",

	// cpu
	UnimplementedInstruction:       "cpu error: unimplemented instruction (%#02x) at (%#04x)",
	InvalidOpcode:                  "cpu error: invalid opcode (%#04x)",
	InvalidResult:                  "cpu error: %v",
	ProgramCounterCycled:           "cpu error: program counter cycled back to 0x0000",
	InvalidOperationMidInstruction: "cpu error: invalid operation mid-instruction (%v)",

	// memory
	MemoryError:       "memory error: %v",
	UnreadableAddress: "memory error: memory location is not readable (%#04x)",
	UnwritableAddress: "memory error: memory location is not writable (%#04x)",
	UnpokeableAddress: "memory error: cannot poke address (%v)",
	UnpeekableAddress: "memory error: cannot peek address (%v)",

	// cartridges
	CartridgeError:   "cartridge error: %v",
	CartridgeEjected: "cartridge error: no cartridge attached",

	// peripherals
	PeriphHardwareUnavailable: "peripheral error: controller hardware unavailable (%v)",
	UnknownPeriphEvent:        "peripheral error: %v: unsupported event (%v)",

	// television
	UnknownTVRequest: "television error: unsupported request (%v)",
	Television:       "television error: %v",

	// screen digest
	ScreenDigest: "television error: screendigest: %v",

	// audio2wav
	WavWriter: "wav writer: %v",

	// gui
	UnsupportedGUIRequest: "gui error: unsupported request (%v)",
	SDL:                   "SDL: %v",
}
