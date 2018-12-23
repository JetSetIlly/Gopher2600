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

	// TV
	UnknownTVRequest
	SDLTV
	ImageTV
	DigestTV

	// Peripherals
	NoControllersFound
)
