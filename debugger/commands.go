package debugger

import "gopher2600/debugger/parser"

// debugger keywords. not a useful data structure but we can use these to form
// the more useful DebuggerCommands and Help structures
const (
	KeywordHelp          = "HELP"
	KeywordInsert        = "INSERT"
	KeywordSymbol        = "SYMBOL"
	KeywordBreak         = "BREAK"
	KeywordTrap          = "TRAP"
	KeywordList          = "LIST"
	KeywordClear         = "CLEAR"
	KeywordOnHalt        = "ONHALT"
	KeywordOnStep        = "ONSTEP"
	KeywordLast          = "LAST"
	KeywordMemMap        = "MEMMAP"
	KeywordQuit          = "QUIT"
	KeywordReset         = "RESET"
	KeywordRun           = "RUN"
	KeywordStep          = "STEP"
	KeywordStepMode      = "STEPMODE"
	KeywordTerse         = "TERSE"
	KeywordVerbose       = "VERBOSE"
	KeywordVerbosity     = "VERBOSITY"
	KeywordDebuggerState = "DEBUGGERSTATE"
	KeywordCPU           = "CPU"
	KeywordPeek          = "PEEK"
	KeywordRAM           = "RAM"
	KeywordRIOT          = "RIOT"
	KeywordTIA           = "TIA"
	KeywordTV            = "TV"
	KeywordPlayer        = "PLAYER"
	KeywordMissile       = "MISSILE"
	KeywordBall          = "BALL"
	KeywordPlayfield     = "PLAYFIELD"
	KeywordDisplay       = "DISPLAY"
	KeywordScript        = "SCRIPT"
	KeywordDisassemble   = "DISASSEMBLE"

	SubKeywordBreaks = "BREAKS"
	SubKeywordTraps  = "TRAPS"
	SubKeywordVideo  = "VIDEO"
	SubKeywordCPU    = "CPU"
)

// DebuggerCommands provides:
//	- the list of debugger commands (keys to the map)
//  - the tab completion method for each argument for each command
var DebuggerCommands = parser.Commands{
	KeywordInsert:        parser.CommandArgs{parser.Arg{Typ: parser.ArgFile, Req: true}},
	KeywordSymbol:        parser.CommandArgs{parser.Arg{Typ: parser.ArgString, Req: true}},
	KeywordBreak:         parser.CommandArgs{parser.Arg{Typ: parser.ArgTarget, Req: true}, parser.Arg{Typ: parser.ArgValue, Req: false}},
	KeywordTrap:          parser.CommandArgs{parser.Arg{Typ: parser.ArgTarget, Req: true}},
	KeywordList:          parser.CommandArgs{parser.Arg{Typ: parser.ArgKeyword, Req: true, Vals: parser.Keywords{SubKeywordBreaks, SubKeywordTraps}}},
	KeywordClear:         parser.CommandArgs{parser.Arg{Typ: parser.ArgKeyword, Req: true, Vals: parser.Keywords{SubKeywordBreaks, SubKeywordTraps}}},
	KeywordOnHalt:        parser.CommandArgs{parser.Arg{Typ: parser.ArgIndeterminate}},
	KeywordOnStep:        parser.CommandArgs{parser.Arg{Typ: parser.ArgIndeterminate}},
	KeywordLast:          parser.CommandArgs{},
	KeywordMemMap:        parser.CommandArgs{},
	KeywordQuit:          parser.CommandArgs{},
	KeywordReset:         parser.CommandArgs{},
	KeywordRun:           parser.CommandArgs{},
	KeywordStep:          parser.CommandArgs{},
	KeywordStepMode:      parser.CommandArgs{parser.Arg{Typ: parser.ArgKeyword, Req: false, Vals: parser.Keywords{SubKeywordCPU, SubKeywordVideo}}},
	KeywordTerse:         parser.CommandArgs{},
	KeywordVerbose:       parser.CommandArgs{},
	KeywordVerbosity:     parser.CommandArgs{},
	KeywordDebuggerState: parser.CommandArgs{},
	KeywordCPU:           parser.CommandArgs{},
	KeywordPeek:          parser.CommandArgs{parser.Arg{Typ: parser.ArgValue | parser.ArgString, Req: true}, parser.Arg{Typ: parser.ArgIndeterminate}},
	KeywordRAM:           parser.CommandArgs{},
	KeywordRIOT:          parser.CommandArgs{},
	KeywordTIA:           parser.CommandArgs{},
	KeywordTV:            parser.CommandArgs{},
	KeywordPlayer:        parser.CommandArgs{},
	KeywordMissile:       parser.CommandArgs{},
	KeywordBall:          parser.CommandArgs{},
	KeywordPlayfield:     parser.CommandArgs{},
	KeywordDisplay:       parser.CommandArgs{},
	KeywordScript:        parser.CommandArgs{parser.Arg{Typ: parser.ArgFile, Req: true}},
	KeywordDisassemble:   parser.CommandArgs{},
}

func init() {
	// add the help command. we can't add the complete definition for the
	// command in the DebuggerCommands declaration because the list of Keywords
	// refers to DebuggerCommands itself
	DebuggerCommands[KeywordHelp] = parser.CommandArgs{parser.Arg{Typ: parser.ArgKeyword, Req: false, Vals: &DebuggerCommands}}
}

// Help contains the help text for the debugger's top level commands
var Help = map[string]string{
	KeywordHelp:          "Lists commands and provides help for individual debugger commands",
	KeywordInsert:        "Insert cartridge into emulation (from file)",
	KeywordSymbol:        "Search for the address label symbol in disassembly. returns address",
	KeywordBreak:         "Cause emulator to halt when conditions are met",
	KeywordList:          "List current entries for BREAKS and TRAPS",
	KeywordClear:         "Clear all entries in BREAKS and TRAPS",
	KeywordTrap:          "Cause emulator to halt when specified machine component is touched",
	KeywordOnHalt:        "Commands to run whenever emulation is halted (separate commands with comma)",
	KeywordOnStep:        "Commands to run whenever emulation steps forward an cpu/video cycle (separate commands with comma)",
	KeywordLast:          "Prints the result of the last cpu/video cycle",
	KeywordMemMap:        "Display high-levl VCS memory map",
	KeywordQuit:          "Exits the emulator",
	KeywordReset:         "Rest the emulation to its initial state",
	KeywordRun:           "Run emulator until next halt state",
	KeywordStep:          "Step forward emulator one step (see STEPMODE command)",
	KeywordStepMode:      "Change method of stepping: CPU or VIDEO",
	KeywordTerse:         "Use terse format when displaying machine information",
	KeywordVerbose:       "Use verbose format when displaying machine information",
	KeywordVerbosity:     "Display which fomat is used when displaying machine information (see TERSE and VERBOSE commands)",
	KeywordDebuggerState: "Display summary of debugger options",
	KeywordCPU:           "Display the current state of the CPU",
	KeywordPeek:          "Inspect an individual memory address",
	KeywordRAM:           "Display the current contents of PIA RAM",
	KeywordRIOT:          "Display the current state of the RIOT",
	KeywordTIA:           "Display current state of the TIA",
	KeywordTV:            "Display the current TV state",
	KeywordPlayer:        "Display the current state of the Player 0/1 sprite",
	KeywordMissile:       "Display the current state of the Missile 0/1 sprite",
	KeywordBall:          "Display the current state of the Ball sprite",
	KeywordPlayfield:     "Display the current playfield data",
	KeywordDisplay:       "Display the TV image",
	KeywordScript:        "Run commands from specified file",
	KeywordDisassemble:   "Print the full cartridge disassembly",
}
