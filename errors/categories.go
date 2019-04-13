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
	ScriptFileCannotOpen
	ScriptFileError
	ScriptRunError
	ScriptEnd

	// Capture
	ScriptRecordError

	// Regression
	RegressionEntryExists
	RegressionEntryCollision
	RegressionEntryDoesNotExist
	RegressionEntryFail

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
	ImageTV
	DigestTV

	// Controllers
	StickDisconnected

	// GUI
	UnknownGUIRequest
	SDL

	// Peripherals
	NoControllersFound
)
