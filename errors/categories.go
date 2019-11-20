package errors

// list of error numbers
const (
	// PanicErrors should be used only as an alternative to panic(). that is
	// errors where there is no good response beyond suggesting that a terrible
	// mistake has been made. PanicErrors should be treated like actual
	// panic()s and cause the program (or the sub-system) to cease as soon as
	// possible.
	//
	// if is not practical to cause the program to cease then at the very
	// least, the PanicError should result in the display of the error message
	// in big, friendly letters.
	//
	// actual panic()s should only be used when the mistake is so heinous that
	// it suggests a fundamental misunderstanding has taken place and so, as it
	// were, all bets are off.
	PanicError Errno = iota

	// sentinal
	UserInterrupt
	UserSuspend
	ScriptEnd
	PowerOff
	PeriphUnplugged
	TVOutOfSpec

	// program modes
	PlayError
	DebuggerError
	DisasmError
	PerformanceError

	// debugger
	ParserError
	ValidationError
	InvalidTarget
	CommandError
	TerminalError
	GUIEventError
	ReflectionNotRunning

	// script
	ScriptScribeError
	ScriptFileUnavailable
	ScriptFileError
	ScriptRunError

	// recorder
	RecordingError
	PlaybackError
	PlaybackHashError

	// database
	DatabaseError
	DatabaseSelectEmpty
	DatabaseKeyError
	DatabaseFileUnavailable

	// regression
	RegressionError
	RegressionFrameError
	RegressionPlaybackError

	// setup
	SetupError
	SetupPanelError

	// symbols
	SymbolsFileUnavailable
	SymbolsFileError
	SymbolUnknown

	// cartridgeloader
	CartridgeLoader

	// vcs
	VCSError

	// cpu
	UnimplementedInstruction
	InvalidOpcode
	InvalidResult
	ProgramCounterCycled
	InvalidOperationMidInstruction

	// memory
	MemoryError
	UnreadableAddress
	UnwritableAddress
	UnpokeableAddress
	UnpeekableAddress
	UnrecognisedAddress

	// cartridges
	CartridgeError
	CartridgeEjected

	// peripherals
	PeriphHardwareUnavailable
	UnknownPeriphEvent

	// tv
	UnknownTVRequest
	Television

	// screen digest
	ScreenDigest

	// gui
	UnsupportedGUIRequest
	SDL
)
