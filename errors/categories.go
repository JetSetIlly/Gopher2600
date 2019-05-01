package errors

// list of error numbers
const (
	// Debugger
	UserInterrupt Errno = iota
	CommandError
	InvalidTarget

	// Symbols
	SymbolsFileCannotOpen
	SymbolsFileError
	SymbolUnknown

	// Script
	ScriptRecordingError
	ScriptFileCannotOpen
	ScriptFileError
	ScriptRunError
	ScriptEnd

	// Regression
	RegressionDBError
	RegressionFail

	// CPU
	UnimplementedInstruction
	InvalidOpcode
	ProgramCounterCycled
	InvalidOperationMidInstruction

	// Memory
	UnservicedChipWrite
	UnknownRegisterName
	UnreadableAddress
	UnwritableAddress
	UnrecognisedAddress
	UnPokeableAddress

	// Cartridges
	CartridgeFileError
	CartridgeUnsupported
	CartridgeMissing
	CartridgeNoSuchBank

	// TV
	UnknownTVRequest
	BasicTelevision
	ImageTV
	DigestTV

	// GUI
	UnknownGUIRequest
	SDL

	// Peripherals
	NoControllerHardware
	NoPlayerPort
	ControllerUnplugged
	UnknownPeripheralEvent
	PowerOff

	// Recorder
	RecordingError
	PlaybackInvalidFile
	PlaybackError
	PlaybackHashError
)
