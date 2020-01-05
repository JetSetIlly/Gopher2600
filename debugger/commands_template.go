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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package debugger

// debugger keywords
const (
	cmdReset = "RESET"
	cmdQuit  = "QUIT"

	cmdRun     = "RUN"
	cmdStep    = "STEP"
	cmdQuantum = "QUANTUM"
	cmdScript  = "SCRIPT"

	cmdInsert      = "INSERT"
	cmdCartridge   = "CARTRIDGE"
	cmdPatch       = "PATCH"
	cmdDisassembly = "DISASSEMBLY"
	cmdGrep        = "GREP"
	cmdSymbol      = "SYMBOL"
	cmdOnHalt      = "ONHALT"
	cmdOnStep      = "ONSTEP"
	cmdLast        = "LAST"
	cmdMemMap      = "MEMMAP"
	cmdCPU         = "CPU"
	cmdPeek        = "PEEK"
	cmdPoke        = "POKE"
	cmdRAM         = "RAM"
	cmdTimer       = "TIMER"
	cmdTIA         = "TIA"
	cmdAudio       = "AUDIO"
	cmdTV          = "TV"
	cmdPlayer      = "PLAYER"
	cmdMissile     = "MISSILE"
	cmdBall        = "BALL"
	cmdPlayfield   = "PLAYFIELD"
	cmdDisplay     = "DISPLAY"

	// user input
	cmdPanel = "PANEL"
	cmdStick = "STICK"

	// halt conditions
	cmdBreak = "BREAK"
	cmdTrap  = "TRAP"
	cmdWatch = "WATCH"
	cmdList  = "LIST"
	cmdDrop  = "DROP"
	cmdClear = "CLEAR"
)

const cmdHelp = "HELP"

var commandTemplate = []string{
	cmdReset,
	cmdQuit,

	cmdRun,
	cmdStep + " (CPU|VIDEO|%<target>S)",
	cmdQuantum + " (CPU|VIDEO)",
	cmdScript + " [RECORD %<new file>S|END|%<file>S]",

	cmdInsert + " %<cartridge>F",
	cmdCartridge + " (ANALYSIS|BANK %<number>N)",
	cmdPatch + " %<patch file>S",
	cmdDisassembly + " (BYTECODE) (%<bank num>N)",
	cmdGrep + " (MNEMONIC|OPERAND) %<search>S",
	cmdSymbol + " [%<symbol>S (ALL|MIRRORS)|LIST (LOCATIONS|READ|WRITE)]",
	cmdOnHalt + " (OFF|ON|%<command>S {%<commands>S})",
	cmdOnStep + " (OFF|ON|%<command>S {%<commands>S})",
	cmdLast + " (DEFN|BYTECODE)",
	cmdMemMap,
	cmdCPU + " (SET [PC|A|X|Y|SP] [%<register value>N]|BUG (ON|OFF))",
	cmdPeek + " [%<address>S] {%<addresses>S}",
	cmdPoke + " %<address>S [%<value>N] {%<values>N}",
	cmdRAM + " (CART)",
	cmdTimer,
	cmdTIA + " (DELAYS)",
	cmdAudio,
	cmdTV + " (SPEC)",
	cmdPlayer + " (0|1)",
	cmdMissile + " (0|1)",
	cmdBall,
	cmdPlayfield,
	cmdDisplay + " (ON|OFF|MASK|UNMASK|SCALE [%<scale value>P]|ALT (ON|OFF)|OVERLAY (ON|OFF))", // see notes

	// user input
	cmdPanel + " (SET [P0PRO|P1PRO|P0AM|P1AM|COL|BW]|TOGGLE [P0|P1|COL])",
	cmdStick + " [0|1] [LEFT|RIGHT|UP|DOWN|FIRE|NOLEFT|NORIGHT|NOUP|NODOWN|NOFIRE]",

	// halt conditions
	cmdBreak + " [%<target>S %<value>N|%<pc value>N] {& %<target>S %<value>N|& %<value>N}",
	cmdTrap + " [%<target>S] {%<targets>S}",
	cmdWatch + " (READ|WRITE) [%<address>S] (%<value>S)",
	cmdList + " [BREAKS|TRAPS|WATCHES|ALL]",
	cmdDrop + " [BREAK|TRAP|WATCH] %<number in list>N",
	cmdClear + " [BREAKS|TRAPS|WATCHES|ALL]",
}

// list of commands that should not be executed when recording/playing scripts
var scriptUnsafeTemplate = []string{
	cmdScript + " [RECORD %S]",
	cmdRun,
}
