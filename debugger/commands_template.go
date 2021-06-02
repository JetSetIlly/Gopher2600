// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package debugger

// debugger keywords.
const (
	cmdReset = "RESET"
	cmdQuit  = "QUIT"

	cmdRun     = "RUN"
	cmdStep    = "STEP"
	cmdHalt    = "HALT"
	cmdQuantum = "QUANTUM"
	cmdScript  = "SCRIPT"
	cmdRewind  = "REWIND"

	cmdInsert    = "INSERT"
	cmdCartridge = "CARTRIDGE"
	cmdPatch     = "PATCH"
	cmdDisasm    = "DISASM"
	cmdLint      = "LINT"
	cmdGrep      = "GREP"
	cmdSymbol    = "SYMBOL"
	cmdOnHalt    = "ONHALT"
	cmdOnStep    = "ONSTEP"
	cmdOnTrace   = "ONTRACE"
	cmdLast      = "LAST"
	cmdMemMap    = "MEMMAP"
	cmdCPU       = "CPU"
	cmdPeek      = "PEEK"
	cmdPoke      = "POKE"
	cmdRAM       = "RAM"
	cmdTIA       = "TIA"
	cmdRIOT      = "RIOT"
	cmdAudio     = "AUDIO"
	cmdTV        = "TV"
	cmdPlayer    = "PLAYER"
	cmdMissile   = "MISSILE"
	cmdBall      = "BALL"
	cmdPlayfield = "PLAYFIELD"
	cmdDisplay   = "DISPLAY"

	cmdPlusROM = "PLUSROM"

	// user input.
	cmdController = "CONTROLLER"
	cmdPanel      = "PANEL"
	cmdStick      = "STICK"
	cmdKeyboard   = "KEYBOARD"

	// halt conditions.
	cmdBreak = "BREAK"
	cmdTrap  = "TRAP"
	cmdWatch = "WATCH"
	cmdTrace = "TRACE"
	cmdList  = "LIST"
	cmdDrop  = "DROP"
	cmdClear = "CLEAR"

	// meta.
	cmdPrefs    = "PREFS"
	cmdLog      = "LOG"
	cmdMemUsage = "MEMUSAGE"
)

const cmdHelp = "HELP"

var commandTemplate = []string{
	cmdReset,
	cmdQuit,

	cmdRun,
	cmdStep + " (BACK) (INSTRUCTION|VIDEO|SCANLINE|FRAME)",
	cmdHalt,
	cmdQuantum + " (INSTRUCTION|VIDEO)",
	cmdScript + " [RECORD %<new file>F|END|%<file>F]",
	cmdRewind + " [%<frame>N|LAST|SUMMARY]",

	cmdInsert + " %<cartridge>F",
	cmdCartridge + " (BANK|MAPPING|HASH|STATIC|REGISTERS|RAM|HOTLOAD)",
	cmdPatch + " %<patch file>S",
	cmdDisasm + " (BYTECODE) (%<bank>N|%<address>N)",
	cmdLint,
	cmdGrep + " (OPERATOR|OPERAND) %<search>S",
	cmdSymbol + " [LIST (LABELS|READ|WRITE)|%<symbol>S]",
	cmdOnHalt + " (OFF|ON|%<command>S {%<commands>S})",
	cmdOnStep + " (OFF|ON|%<command>S {%<commands>S})",
	cmdOnTrace + " (OFF|ON|%<command>S {%<commands>S})",
	cmdLast + " (DEFN|BYTECODE)",
	cmdMemMap + " (%<address>S)",
	cmdCPU + " (STATUS ([SET|UNSET|TOGGLE] [S|O|B|D|I|Z|C])|(SET [PC|A|X|Y|SP] [%<register value>S]))",
	cmdPeek + " [%<address>S] {%<addresses>S}",
	cmdPoke + " %<address>S [%<value>N] {%<values>N}",
	cmdRAM,
	cmdTIA + " (HMOVE)",
	cmdRIOT + " (PORTS|TIMER)",
	cmdAudio,
	cmdTV + " (SPEC (NTSC|PAL|PAL60|AUTO))",
	cmdPlayer + " (0|1)",
	cmdMissile + " (0|1)",
	cmdBall,
	cmdPlayfield,
	cmdDisplay + " [ON|OFF]",

	// peripherals (components that might not be present)
	cmdPlusROM + " (NICK [%<name>S]|ID [%<id>S]|HOST [%<host>S]|PATH [%<path>S])",

	// user input
	cmdController + " [LEFT|RIGHT] (AUTO|STICK|PADDLE|KEYBOARD)",
	cmdPanel + " (SET [P0PRO|P1PRO|P0AM|P1AM|COL|BW]|TOGGLE [P0|P1|COL]|[HOLD|RELEASE] [SELECT|RESET])",
	cmdStick + " [LEFT|RIGHT] [LEFT|RIGHT|UP|DOWN|FIRE|NOLEFT|NORIGHT|NOUP|NODOWN|NOFIRE]",
	cmdKeyboard + " [LEFT|RIGHT] [NONE|0|1|2|3|4|5|6|7|8|9|*|#]",

	// halt conditions
	cmdBreak + " [%<pc value>S|%<target>S %<value>N] {& %<value>S|%<target>S %<value>S}",

	cmdTrap + " [%<target>S] {%<targets>S}",
	cmdWatch + " (READ|WRITE) (STRICT) [%<address>S] (%<value>S)",
	cmdTrace + " (%<address>S)",
	cmdList + " [BREAKS|TRAPS|WATCHES|TRACES|ALL]",
	cmdDrop + " [BREAK|TRAP|WATCH|TRACE] %<number in list>N",
	cmdClear + " [BREAKS|TRAPS|WATCHES|TRACES|ALL]",

	// emulation
	cmdPrefs + " ([LOAD|SAVE]|[SET|UNSET|TOGGLE] [FXXXMIRROR|SYMBOLS]|REWIND [MAX %<entries>N|FREQ %<frames>N])",
	cmdLog + " (LAST|RECENT|CLEAR)",
	cmdMemUsage,
}

// list of commands that should not be executed when recording/playing scripts.
var scriptUnsafeTemplate = []string{
	cmdScript + " [RECORD %S]",
	cmdRun,
}
