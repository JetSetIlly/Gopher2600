package debugger

// Help contains the help text for the debugger's top level commands
var help = map[string]string{
	cmdHelp:  "Lists commands and provides help for individual debugger commands",
	cmdReset: "Reset the emulation to its initial state",
	cmdQuit:  "Exits the emulator",
	cmdExit:  "Exits the emulator",

	cmdRun:         "Run emulator until next halt state",
	cmdStep:        "Step forward one step. Optional argument sets the amount to step by (eg. frame, scanline, etc.)",
	cmdGranularity: "Change method of stepping: CPU or VIDEO",
	cmdScript:      "Run commands from specified file or record commands to a file",

	cmdInsert:      "Insert cartridge into emulation (from file)",
	cmdDisassembly: "Print the full cartridge disassembly",
	cmdGrep:        "Simple string search (case insensitive) of the disassembly",
	cmdSymbol:      "Search for the address label symbol in disassembly. returns address",
	cmdOnHalt:      "Commands to run whenever emulation is halted (separate commands with comma)",
	cmdOnStep:      "Commands to run whenever emulation steps forward an cpu/video cycle (separate commands with comma)",
	cmdLast:        "Prints the result of the last cpu/video cycle",
	cmdMemMap:      "Display high-level VCS memory map",
	cmdCartridge:   "Display information about the current cartridge",
	cmdCPU:         "Display the current state of the CPU",
	cmdPeek:        "Inspect an individual memory address",
	cmdPoke:        "Modify an individual memory address",
	cmdPatch:       "Apply a patch file to the loaded cartridge",
	cmdHexLoad:     "Modify a sequence of memory addresses. Starting address must be numeric.",
	cmdRAM:         "Display the current contents of PIA RAM",
	cmdRIOT:        "Display the current state of the RIOT",
	cmdTIA:         "Display current state of the TIA",
	cmdTV:          "Display the current TV state",
	cmdPanel:       "Inspect front panel settings",
	cmdPlayer:      "Display the current state of the player 0/1 sprite",
	cmdMissile:     "Display the current state of the missile 0/1 sprite",
	cmdBall:        "Display the current state of the ball sprite",
	cmdPlayfield:   "Display the current playfield data",
	cmdDisplay:     "Display the TV image",
	cmdStick:       "Emulate a joystick input for Player 0 or Player 1",

	// halt conditions
	cmdBreak: "Cause emulator to halt when conditions are met",
	cmdTrap:  "Cause emulator to halt when specified machine component is touched",
	cmdWatch: "Watch a memory address for activity",
	cmdList:  "List current entries for breaks, traps and watches",
	cmdDrop:  "Drop a specific break, trap or watch condition, using the number of the condition reported by LIST",
	cmdClear: "Clear all breaks, traps and watches",
}
