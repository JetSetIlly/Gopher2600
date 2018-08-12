package errors

var messages = map[Errno]string{
	// Debugger
	InputInvalidCommand:   "%s",
	InputTooManyArgs:      "%s",
	InputTooFewArgs:       "%s",
	InputEmpty:            "%s",
	SymbolsFileCannotOpen: "no symbols file for %s",
	SymbolsFileError:      "error processing symbols file (%s)",
	SymbolUnknown:         "unrecognised symbol (%s)",
	ScriptFileCannotOpen:  "cannot open script file (%s)",
	InvalidTarget:         "invalid target (%s)",

	// CPU
	UnimplementedInstruction: "unimplemented instruction (%0#x) at (%#04x)",
	NullInstruction:          "unimplemented instruction (0xff)",
	ProgramCounterCycled:     "program counter cycled back to 0x0000",

	// Memory
	UnservicedChipWrite: "chip memory write signal has not been serviced since previous write (%s)",
	UnknownRegisterName: "can't find register name (%s) in list of read addreses in %s memory",
	UnreadableAddress:   "memory location is not readable (%#04x)",
	UnwritableAddress:   "memory location is not writable (%#04x)",
	UnrecognisedAddress: "address unrecognised (%v)",

	// Cartridges
	CartridgeFileCannotOpen: "cannot open cartridge (%s)",
	CartridgeFileError:      "error reading cartridge file (%s)",
	CartridgeInvalidSize:    "cartridge size is not recognised (%d)",

	// TV
	UnknownTVRequest: "TV does not support %v request",

	// Peripherals
	NoControllersFound: "no controllers found",
}

// more error strings -- these are strings that are used as arguments to error
// string messages
const (
	FileTruncated string = "file truncated"
)
