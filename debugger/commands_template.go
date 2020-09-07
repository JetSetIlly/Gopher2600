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

// debugger keywords
const (
	cmdReset = "RESET"
	cmdQuit  = "QUIT"

	cmdRun     = "RUN"
	cmdStep    = "STEP"
	cmdHalt    = "HALT"
	cmdQuantum = "QUANTUM"
	cmdScript  = "SCRIPT"

	cmdInsert      = "INSERT"
	cmdCartridge   = "CARTRIDGE"
	cmdPatch       = "PATCH"
	cmdDisassembly = "DISASSEMBLY"
	cmdLint        = "LINT"
	cmdGrep        = "GREP"
	cmdSymbol      = "SYMBOL"
	cmdOnHalt      = "ONHALT"
	cmdOnStep      = "ONSTEP"
	cmdOnTrace     = "ONTRACE"
	cmdLast        = "LAST"
	cmdMemMap      = "MEMMAP"
	cmdCPU         = "CPU"
	cmdPeek        = "PEEK"
	cmdPoke        = "POKE"
	cmdRAM         = "RAM"
	cmdTIA         = "TIA"
	cmdRIOT        = "RIOT"
	cmdAudio       = "AUDIO"
	cmdTV          = "TV"
	cmdPlayer      = "PLAYER"
	cmdMissile     = "MISSILE"
	cmdBall        = "BALL"
	cmdPlayfield   = "PLAYFIELD"
	cmdDisplay     = "DISPLAY"

	// user input
	cmdController = "CONTROLLER"
	cmdPanel      = "PANEL"
	cmdStick      = "STICK"
	cmdKeyboard   = "KEYBOARD"

	// halt conditions
	cmdBreak = "BREAK"
	cmdTrap  = "TRAP"
	cmdWatch = "WATCH"
	cmdTrace = "TRACE"
	cmdList  = "LIST"
	cmdDrop  = "DROP"
	cmdClear = "CLEAR"

	// meta
	cmdPref = "PREF"
	cmdLog  = "LOG"
)

const cmdHelp = "HELP"

var commandTemplate = []string{
	cmdReset,
	cmdQuit,

	cmdRun,
	cmdStep + " (CPU|VIDEO|%<target>S)",
	cmdHalt,
	cmdQuantum + " (CPU|VIDEO)",
	cmdScript + " [RECORD %<new file>F|END|%<file>F]",

	cmdInsert + " %<cartridge>F",
	cmdCartridge + " (BANK|STATIC|REGISTERS|RAM)",
	cmdPatch + " %<patch file>S",
	cmdDisassembly + " (BYTECODE) (%<bank num>N)",
	cmdLint,
	cmdGrep + " (MNEMONIC|OPERAND) %<search>S",
	cmdSymbol + " [%<symbol>S (ALL|MIRRORS)|LIST (LOCATIONS|READ|WRITE)]",
	cmdOnHalt + " (OFF|ON|%<command>S {%<commands>S})",
	cmdOnStep + " (OFF|ON|%<command>S {%<commands>S})",
	cmdOnTrace + " (OFF|ON|%<command>S {%<commands>S})",
	cmdLast + " (DEFN|BYTECODE)",
	cmdMemMap + " (%<address>S)",
	cmdCPU + " (SET [PC|A|X|Y|SP] [%<register value>N])",
	cmdPeek + " [%<address>S] {%<addresses>S}",
	cmdPoke + " %<address>S [%<value>N] {%<values>N}",
	cmdRAM,
	cmdTIA,
	cmdRIOT + " (PORTS|TIMER)",
	cmdAudio,
	cmdTV + " (SPEC (PAL|NTSC|AUTO))",
	cmdPlayer + " (0|1)",
	cmdMissile + " (0|1)",
	cmdBall,
	cmdPlayfield,
	cmdDisplay + " (ON|OFF|SCALE [%<scale value>P]|MASKING (ON|OFF)|DEBUG (ON|OFF)|OVERLAY (ON|OFF))", // see notes

	// user input
	cmdController + " [0|1] (AUTO|STICK|PADDLE|KEYBOARD)",
	cmdPanel + " (SET [P0PRO|P1PRO|P0AM|P1AM|COL|BW]|TOGGLE [P0|P1|COL]|[HOLD|RELEASE] [SELECT|RESET])",
	cmdStick + " [0|1] [LEFT|RIGHT|UP|DOWN|FIRE|NOLEFT|NORIGHT|NOUP|NODOWN|NOFIRE]",
	cmdKeyboard + " [0|1] [none|0|1|2|3|4|5|6|7|8|9|*|#]",

	// halt conditions
	cmdBreak + " [%<target>S %<value>N|%<pc value>S] {& %<target>S %<value>S|& %<value>S}",
	cmdTrap + " [%<target>S] {%<targets>S}",
	cmdWatch + " (READ|WRITE) (MIRRORS|ANY) [%<address>S] (%<value>S)",
	cmdTrace + " (%<address>S)",
	cmdList + " [BREAKS|TRAPS|WATCHES|TRACES|ALL]",
	cmdDrop + " [BREAK|TRAP|WATCH|TRACE] %<number in list>N",
	cmdClear + " [BREAKS|TRAPS|WATCHES|TRACES|ALL]",

	// meta
	cmdPref + " ([LOAD|SAVE]|[SET|UNSET|TOGGLE] [RANDSTART|RANDPINS|FXXXMIRROR])",
	cmdLog + " (LAST|CLEAR)",
}

// list of commands that should not be executed when recording/playing scripts
var scriptUnsafeTemplate = []string{
	cmdScript + " [RECORD %S]",
	cmdRun,
}
