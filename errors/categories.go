package errors

// list of error numbers
const (
	// Debugger
	InputEmpty Errno = iota
	CommandError
	SymbolsFileCannotOpen
	SymbolsFileError
	SymbolUnknown
	ScriptFileCannotOpen
	ScriptFileError
	InvalidTarget

	// CPU
	UnimplementedInstruction
	NullInstruction
	ProgramCounterCycled

	// Memory
	UnservicedChipWrite
	UnknownRegisterName
	UnreadableAddress
	UnwritableAddress
	UnrecognisedAddress

	// Cartridges
	CartridgeFileCannotOpen
	CartridgeFileError
	CartridgeInvalidSize

	// TV
	UnknownTVRequest

	// Peripherals
	NoControllersFound
)
