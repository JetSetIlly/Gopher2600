package errors

var messages = map[Errno]string{
	// Debugger
	InputEmpty:            "input is empty",
	CommandError:          "%s",
	SymbolsFileCannotOpen: "no symbols file for %s",
	SymbolsFileError:      "error processing symbols file (%s)",
	SymbolUnknown:         "unrecognised symbol (%s)",
	ScriptFileCannotOpen:  "cannot open script file (%s)",
	ScriptFileError:       "script error: %s",
	ScriptRunError:        "script error: use of '%s' is not allowed in scripts [%s::%d]",
	InvalidTarget:         "invalid target (%s)",
	CannotRecordState:     "cannot record state: %s",

	// Regression
	RegressionEntryExists:       "entry exists (%s)",
	RegressionEntryCollision:    "ROM hash collision (%s AND %s)",
	RegressionEntryDoesNotExist: "entry missing (%s)",
	RegressionEntryFail:         "screen digest mismatch (%s)",

	// CPU
	UnimplementedInstruction:       "unimplemented instruction (%0#x) at (%#04x)",
	InvalidOpcode:                  "invalid opcode (%#04x)",
	ProgramCounterCycled:           "program counter cycled back to 0x0000",
	InvalidOperationMidInstruction: "invalid operation mid-instruction (%s)",

	// Memory
	UnservicedChipWrite: "chip memory write signal has not been serviced since previous write (%s)",
	UnknownRegisterName: "can't find register name (%s) in list of read addreses in %s memory",
	UnreadableAddress:   "memory location is not readable (%#04x)",
	UnwritableAddress:   "memory location is not writable (%#04x)",
	UnrecognisedAddress: "address unrecognised (%v)",
	UnPokeableAddress:   "address is un-poke-able (%v)",

	// Cartridges
	CartridgeFileError:   "error reading cartridge file (%s)",
	CartridgeUnsupported: "cartridge unsupported (%s)",
	CartridgeMissing:     "no cartridge attached",
	CartridgeNoSuchBank:  "bank out of range (%d) for this cartridge (max=%d)",

	// TV
	UnknownTVRequest: "TV does not support %v request",
	ImageTV:          "ImageTV: %s",
	DigestTV:         "DigestTV: %s",

	// GUI
	UnknownGUIRequest: "GUI does not support %v request",
	SDLTV:             "SDLTV: %s",

	// Peripherals
	NoControllersFound: "no controllers found",
}

// more error strings -- these are strings that are used as arguments to error
// string messages
const (
	FileTruncated string = "file truncated"
)
