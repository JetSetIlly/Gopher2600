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

	cmdRun        = "RUN"
	cmdStep       = "STEP"
	cmdHalt       = "HALT"
	cmdQuantum    = "QUANTUM"
	cmdScript     = "SCRIPT"
	cmdRewind     = "REWIND"
	cmdComparison = "COMPARISON"
	cmdGoto       = "GOTO"

	cmdInsert    = "INSERT"
	cmdCartridge = "CARTRIDGE"
	cmdPatch     = "PATCH"
	cmdDisasm    = "DISASM"
	cmdGrep      = "GREP"
	cmdSymbol    = "SYMBOL"
	cmdOnHalt    = "ONHALT"
	cmdOnStep    = "ONSTEP"
	cmdOnTrace   = "ONTRACE"
	cmdLast      = "LAST"
	cmdMemMap    = "MEMMAP"
	cmdCPU       = "CPU"
	cmdBus       = "BUS"
	cmdPeek      = "PEEK"
	cmdPoke      = "POKE"
	cmdSwap      = "SWAP"
	cmdRAM       = "RAM"
	cmdTIA       = "TIA"
	cmdRIOT      = "RIOT"
	cmdAudio     = "AUDIO"
	cmdTV        = "TV"
	cmdPlayer    = "PLAYER"
	cmdMissile   = "MISSILE"
	cmdBall      = "BALL"
	cmdPlayfield = "PLAYFIELD"

	cmdPlusROM = "PLUSROM"
	cmdCoProc  = "COPROC"
	cmdDWARF   = "DWARF"

	// user input.
	cmdPeripheral = "PERIPHERAL"
	cmdPanel      = "PANEL"
	cmdStick      = "STICK"
	cmdKeypad     = "KEYPAD"

	// halt conditions.
	cmdBreak = "BREAK"
	cmdTrap  = "TRAP"
	cmdWatch = "WATCH"
	cmdTrace = "TRACE"
	cmdList  = "LIST"
	cmdDrop  = "DROP"
	cmdClear = "CLEAR"

	// meta.
	cmdLog      = "LOG"
	cmdMemUsage = "MEMUSAGE"
	cmdVersion  = "VERSION"
)

const cmdHelp = "HELP"

var commandTemplate = []string{
	cmdReset,
	cmdQuit,

	cmdRun,
	cmdStep + " (BACK|OVER) (INSTRUCTION|CLOCK|SCANLINE|FRAME)",
	cmdHalt,
	cmdQuantum + " (INSTRUCTION|CYCLE|CLOCK)",
	cmdScript + " [RECORD %<new file>F|END|%<file>F]",
	cmdRewind + " [%<frame>N|LAST|SUMMARY]",
	cmdComparison + " [%<frame>N|LOCK|UNLOCK]",
	cmdGoto + " [%<clock>N] (%<scanline>N) (%<frame>N)",

	cmdInsert + " %<cartridge>F",
	cmdCartridge + " (PATH|NAME|MAPPER|CONTAINER|MAPPEDBANKS|HASH|STATIC|REGISTERS|RAM|DUMP)",
	cmdPatch + " %<patch file>S",
	cmdDisasm + " (BYTECODE|REDUX)",
	cmdGrep + " (OPERATOR|OPERAND|COPROC) %<search>S",
	cmdSymbol + " [LIST (LABELS|READ|WRITE)|%<symbol>S]",
	cmdOnHalt + " (OFF|ON|%<command>S {%<commands>S})",
	cmdOnStep + " (OFF|ON|%<command>S {%<commands>S})",
	cmdOnTrace + " (OFF|ON|%<command>S {%<commands>S})",
	cmdLast + " (DEFN|BYTECODE)",
	cmdMemMap + " (%<address>S)",
	cmdCPU + " (STATUS ([SET|UNSET|TOGGLE] [S|O|B|D|I|Z|C])|(SET [PC|A|X|Y|SP] [%<register value>S]))",
	cmdBus + " (DETAIL)",
	cmdPeek + " [%<address>S] {%<addresses>S}",
	cmdPoke + " %<address>S [%<value>N] {%<values>N}",
	cmdSwap + " %<address>S %<address>S",
	cmdRAM,
	cmdTIA + " (HMOVE)",
	cmdRIOT + " (PORTS|TIMER)",
	cmdAudio,
	cmdTV + " (SPEC (NTSC|PAL|PAL60|PAL-M|SECAM|AUTO))",
	cmdPlayer + " (0|1)",
	cmdMissile + " (0|1)",
	cmdBall,
	cmdPlayfield,

	// peripherals (components that might not be present)
	cmdPlusROM + " (NICK [%<name>S]|ID [%<id>S]|HOST [%<host>S]|PATH [%<path>S])",
	cmdCoProc + " (ID|LIST [FAULTS|SOURCEFILES|FUNCTIONS]|TOP (%<top>N)|MEM {DUMP {%<area>S}}|REGS %<group>S|SET %<register>N %<value>N|STEP)",
	cmdDWARF + " [FUNCTIONS|GLOBALS|LOCALS {DERIVATION|RANGES|ERROR}|FRAMEBASE {DERIVATION}|LINE %<file:line>S|CALLSTACK|CALLERS %<function>S]",

	// user input
	cmdPeripheral + " ([LEFT|RIGHT] (AUTO|STICK|PADDLE|KEYPAD|GAMEPAD|SAVEKEY|ATARIVOX)|SWAP)",
	cmdPanel + " (SET [P0PRO|P1PRO|P0AM|P1AM|COL|BW]|TOGGLE [P0|P1|COL]|[HOLD|RELEASE] [SELECT|RESET])",
	cmdStick + " [LEFT|RIGHT] [LEFT|RIGHT|UP|DOWN|FIRE|NOLEFT|NORIGHT|NOUP|NODOWN|NOFIRE]",
	cmdKeypad + " [LEFT|RIGHT] [NONE|0|1|2|3|4|5|6|7|8|9|*|#]",

	// halt conditions
	cmdBreak + " [%<pc value>S|%<target>S %<value>N] {& %<value>S|%<target>S %<value>S}",
	cmdTrap + " [%<target>S] {%<targets>S}",
	cmdWatch + " (READ|WRITE) (STRICT) (PHANTOM|GHOST) [%<address>S] (%<value>S)",
	cmdTrace + " (STRICT) (%<address>S)",
	cmdList + " [BREAKS|TRAPS|WATCHES|TRACES|ALL]",
	cmdDrop + " [BREAK|TRAP|WATCH|TRACE] %<number in list>N",
	cmdClear + " [BREAKS|TRAPS|WATCHES|TRACES|ALL]",

	// emulation
	cmdLog + " (LAST|RECENT|CLEAR)",
	cmdMemUsage + " (PROFILE)",
	cmdVersion + " (REVISION)",
}

// list of commands that should not be executed when recording/playing scripts.
var scriptUnsafeTemplate = []string{
	cmdScript + " [RECORD %S]",
	cmdRun,
}
