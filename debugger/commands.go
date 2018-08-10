package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
)

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
)

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

var commandTemplate = input.CommandTemplate{
	KeywordInsert:        "%F",
	KeywordSymbol:        "%V",
	KeywordBreak:         "%*",
	KeywordTrap:          "%*",
	KeywordList:          "[BREAKS|TRAPS]",
	KeywordClear:         "[BREAKS|TRAPS]",
	KeywordOnHalt:        "%*",
	KeywordOnStep:        "%*",
	KeywordLast:          "",
	KeywordMemMap:        "",
	KeywordQuit:          "",
	KeywordReset:         "",
	KeywordRun:           "",
	KeywordStep:          "",
	KeywordStepMode:      "[CPU|VIDEO]",
	KeywordTerse:         "",
	KeywordVerbose:       "",
	KeywordVerbosity:     "",
	KeywordDebuggerState: "",
	KeywordCPU:           "",
	KeywordPeek:          "%*",
	KeywordRAM:           "",
	KeywordRIOT:          "",
	KeywordTIA:           "",
	KeywordTV:            "",
	KeywordPlayer:        "",
	KeywordMissile:       "",
	KeywordBall:          "",
	KeywordPlayfield:     "",
	KeywordDisplay:       "[|OFF]",
	KeywordScript:        "%F",
	KeywordDisassemble:   "",
}

// DebuggerCommands is the tree of valid commands
var DebuggerCommands input.Commands

func init() {
	var err error

	// parse command template
	DebuggerCommands, err = input.CompileCommandTemplate(commandTemplate, KeywordHelp)
	if err != nil {
		panic(fmt.Errorf("error compiling command template: %s", err))
	}
}
