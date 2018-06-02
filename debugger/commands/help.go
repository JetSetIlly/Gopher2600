package commands

// Help contains the help text for the debugger's top level commands
var Help = map[string]string{
	KeywordHelp:          "Lists commands and provides help for individual debugger commands",
	KeywordInsert:        "Insert cartridge into emulation (from file)",
	KeywordBreak:         "Cause emulator to halt when conditions are met",
	KeywordTrap:          "Cause emulator to halt when specified machine component is touched",
	KeywordOnHalt:        "Commands to run whenever emulation is halted (separate commands with comma)",
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
	KeywordRIOT:          "Display the current state of the RIOT",
	KeywordTIA:           "Display current state of the TIA",
	KeywordTV:            "Display the current TV state",
	KeywordBall:          "Display the current state of the Ball sprite",
	KeywordDisplay:       "Display the TV image",
	KeywordScript:        "Run commands from specified file"}
