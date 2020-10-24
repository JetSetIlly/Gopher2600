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

var helps = map[string]string{
	cmdHelp: "Lists commands and provides help for individual commands.",

	cmdReset: `Reset the emulated machine (including television) to its initial state. The
debugger itself (breakpoints, etc.) will not be reset.`,

	cmdQuit: `Quit the debugger. If script is being recorded then QUIT will instead halt
recording of the script and not cause the debugger to exit.`,

	cmdRun: `Run emulator until next halt state. A halt state is one triggered by either
a BREAK, TRAP or WATCH condition.`,

	cmdHalt: `Halt emulation. Does nothing if emulation is already halted.`,

	cmdStep: `Step forward one emulation quantum. An optional argument to the STEP command
changes the current quantum and steps forward by one. Permitted quantum values
are CPU and VIDEO. See the help for the QUANTUM command for an explanation.

By way of convenience, the STEP command also accepts a target argument. Let's
call them target- steps.Targets are explained in the help for the BREAK
command.

	STEP FRAME

In the above example, the emulation will run until the next frame is reached.
Think of target stepping as a single use trap. Note that breakpoints, watches
and traps still trigger a halt during a target step.`,

	cmdQuantum: `Change or view stepping quantum. The stepping quantum defines the frequency
at which the emulation is checked and reported upon by the debugger.

There are two quantum modes. The CPU quantum mode causes the debugger to step
one CPU instruction at a time, regardless of how many cycles the instruction
takes. 

The VIDEO quantum mode meanwhile, causes the debugger to step one video cycle
at a time. Compared to the CPU quantum mode, the VIDEO quantum is more uniform
but is inherently slower because of the increased number of BREAK, TRAP
and WATCH checks performed by the debugger.`,

	cmdScript: `Run commands from specified file or record commands to a file. The RECORD
argument indicates that a new script is to be recorded. Recording will not
start if the script file already exists.

Some commands are disallowed in scripts. In particular, you cannot RUN in a
script (but you can STEP). Also, you cannot record a new script during another
script operation but you can playback a script while recording.

The debugger prompt will show that a script recording is taking place.
Recording is halted with a call to QUIT or an interrupt signal (the Ctrl-C
keypress). The quit event itself will not be recorded in the script. Manually
including the QUIT command in a script however, will cause the debugger to
exit.

When manually writing a script in text editor it is sometimes useful to write
comments.  Comments are line oriented and are indicated by the # character.`,

	cmdTimeline: `Move timeline to the numbered frame or to CURRENT.`,

	cmdInsert: `Insert cartridge into emulation. Cartridge names (with paths) beginning with
http:// will loaded via the http protocol. If no such protocol is present, the
cartridge will be loaded from disk.`,

	cmdCartridge: `Display information about the current cartridge. Without arguments the command
will show where the game was loaded from, the cartridge type and bank number. The BANK
argument meanwhile can be used to switch banks (if possible).`,

	cmdPatch: "Apply a patch file to the loaded cartridge",

	cmdDisassembly: `Display cartridge disassembly. By default, all banks will be displayed. Single
banks can be displayed by specifying the bank number. Use BYTECODE to display raw bytes alongside
the disassembly.`,

	cmdGrep: `Simple string search (case insensitive) of the disassembly. Prints all matching lines
in the disassembly to the termain.

The scope of the GREP can be restricted to the MNEMONIC and OPERAND columns. By
default GREP will consider the entire line.`,

	cmdSymbol: `The SYMBOL command has two modes of operation. The first mode returns the address of
the specified symbol. For example:

	SYMBOL CXM1P

Will return:

	CXM1P (read) -> 0x0001

This tells us that the cxm0p symbol is recognised, is a symbol for a read
address and referes to address 0x0001. Many symbols point to addresses that are
mirrored. You can quickly see this with the MIRRORS or ALL argument.

	SYMBOL CXM1P MIRRORS

The above example will return every address that mirrors the primary address.

The second mode of operation allows you to view all the symbols in each symbol
table. There are three symbol tables: READ, WRITE and LABEL.

Note that a cartridge without an accompanying symbols file will only have the
canonical Atari VCS symbols defined and possibly symbols associated with a
particular cartridge type.`,

	cmdOnHalt: `Define commands to run whenever emulation is halted. A halt is
caused by a BREAK, a TRAP, a WATCH or a manual interrupt. Specify multiple
commands by separating with a comma.

THE OFF argument can be used to toggle the ONHALT commands temporarily. Use the
ON argument to resume ONSTEP reporting.`,

	cmdOnStep: `Define commands to run whenever emulation moves forward one step. A step
is defined by the QUANTUM command. Specify multiple commands by separating with
a comma.

THE OFF argument can be used to toggle the ONSTEP commands temporarily. Use the
ON argument to resume ONSTEP reporting.`,

	cmdOnTrace: `Define commands to run whenever a trace condition is met. Unlike the ONSTEP
and ONHALT commands there is no OFF argument.`,

	cmdLast: `Prints the disassembly of the last cpu/video cycle. Use the BYTECODE argument 
to display the raw bytes alongside the disassembly. The DEFN argument meanwhile
will display the definition of the opcode that was used during execution.`,

	cmdMemMap: `Display high-level VCS memory map. With the optional address argument information
about the address will be displayed.`,

	cmdCPU: `Display the current state of the CPU. The SET argument can be used to change the
contents of the CPU registers.`,

	cmdPeek: `Inspect memory addresses for content. Addresses can be specified by symbolically
or numerically.`,

	cmdPoke: `Modify an individual memory address. Addresses can be specified symbolically
or numerically. Mulptiple data values will be poked into consecutive addresses.`,

	cmdRAM: `Display the current contents of RAM. The optional CART argument will display any
additional RAM in the cartridge.`,

	cmdTIA: `Display current state of the TIA video signal:

        111011 (09) _.--*__.--._ 39 13.0

            |             |       |  |
            |             |       |  |
    polycounter           |       |  |
                          |       |  |
       phaseclock --------+       |  |
                                  |  |
           video cycles ----------+  |
                                     |
               cpu cycles -----------+

Video and CPU cycles are counted from the beginning of the current scanline.`,

	cmdRIOT: `Display current state of the RIOT. Without an argument the command will display
information about the RIOT ports (SWCHA, etc.)`,

	cmdAudio: `Display the current state of the audio subsystem.

        ch0: 0000 @ 00100 ^ 0100  ch1: 0000 @ 10000 ^ 0100

              |       |       |
    control --+       |       |
                      |       |
       frequency -----+       |
                              |
           volume ------------+`,

	cmdTV: `Display the current TV state. Optional argument SPEC will display the currently
selected TV specification. Supplying an argument to the TV SPEC command will set the TV to that
specification. AUTO indicates that the specification will change if the condition of the TV signal
suggest that it should.`,

	cmdPlayer: `Display the current state of the player sprites. The player information to
display can be selected with 0 or 1 arguments. Omitting this argument will show
information for both players.

        player0: 101100 (36) _.--.__*--._ [021 > 0x0 > 016] | vdel

           |           |           |              |         |   |
           |           |           |              |         |   |
      sprite id        |           |              |         |   |
                       |           |              |         |   |
       polycounter-----+           |              |         |   |
                                   |              |         |   |
          phaseclock --------------+              |         |   |
                                                  |         |   |
        position > move value > new position -----+         |   |
                                                            |   |
             sizing / copy value (NUSIZ) -------------------+   |
                                                                |
                   notes ---------------------------------------+

The NUSIZ column can take the following forms:

        |                 Single copy

        |_|               Two copies (close)

        |__|              Two copies (medium)

        |_|_|             Three copies (close)

        |___|             Two copies (wide)

        ||                Double width

        |__|__|           Three copies (medium)

        ||||              Quadruple width

The notes column lists transient properties of the sprite: whether the vertical
delay flag is set (as shown in the example); whether the sprite is being drawn
(along with pixel count); which copy is being drawn; whether the player is
reflected; and whether the sprite is currently moving.

Note that these notes apply to the "current" video cycle only. For example, to
say that the sprite is currently moving it is meant the HMOVE process is in
process and has yet to complete. It does not mean the sprite has already moved
or will move later in the frame/scanline.`,

	cmdMissile: `Display the current state of the missile sprites. The missile information to
display can be selected with the 0 or 1 arguments. Omitting this argument will show information
for both missiles.

        missile0: 011101 (30) _*--.__.--._ [002 > 0x0 > 002] | disb

           |           |           |               |         |   |
           |           |           |               |         |   |
      sprite id        |           |               |         |   |
                       |           |               |         |   |
       polycounter-----+           |               |         |   |
                                   |               |         |   |
          phaseclock --------------+               |         |   |
                                                   |         |   |
        position > move value > new position ------+         |   |
                                                             |   |
             copy value (NUSIZ) -----------------------------+   |
                                                                 |
                   notes ----------------------------------------+

The NUSIZ column can take the following forms:

        |                 Single copy

        |_|               Two copies (close)

        |__|              Two copies (medium)

        |_|_|             Three copies (close)

        |___|             Two copies (wide)

        |__|__|           Three copies (medium)

For clarity, the size of the missile is listed in the notes columns: 2x, 4x or
8x.

The notes column is also used to indicate: whether the missile is being drawn;
whether the sprite is disabled (as in the example above); whether the missile
is tracking the player position; or whether the sprite is currently moving.

Note that these notes apply to the "current" video cycle only. For example, to
say that the sprite is currently moving it is meant the HMOVE process is in
process and has yet to complete. It does not mean the sprite has already moved
or will move later in the frame/scanline.`,

	cmdBall: `Display the current state of the ball sprite.

        ball: 011010 (21) _*--.__.--._ [038 > 0x0 > 038] disb

           |        |           |               |         |
           |        |           |               |         |
      sprite id     |           |               |         |
                    |           |               |         |
     polycounter ---+           |               |         |
                                |               |         |
          phaseclock -----------+               |         |
                                                |         |
        position > move value > new position ---+         |
                                                          |
              notes --------------------------------------+

The notes column indicates other details about the current state of the ball
sprite. In addition to noting whether the ball sprite is disabled (as in the
example above) the notes column can also note: whether the vertical delay bit
is set; whether the sprite is currently being drawn; whether the sprite is
currently being moved.

Note that these notes apply to the "current" video cycle only. For example, to
say that the sprite is currently moving it is meant the HMOVE process is in
process and has yet to complete. It does not mean the sprite has already moved
or will move later in the frame/scanline.`,

	cmdPlayfield: `Display the current playfield data.

        playfield: 0110 00101011 01100110 pri

            |        |     |         |     |
            |        |     |         |     |
      id ---+        |     |         |     |
                     |     |         |     |
            pf0 -----+     |         |     |
                           |         |     |
                     pf1 --+         |     |
                                     |     |
                             pf2 ----+     |
                                           |
                                notes -----+

The playfield registers are presented as they are stored. The TIA of course,
reads the bits in a different order but that is not represented here.

The notes field shows the following information as appropriate: priority mode
(as in the example above); scoremode; reflected mode.`,

	cmdDisplay: `Display and otherwise control the TV GUI. The GUI can be shown and hidden
with the ON and OFF arguments.

The MASK and UNMASK arguments toggle the normally invisible areas of the
display: the hblank, vblank and overscan areas of the screen.

An unmasked display is useful when stepping through a ROM because it allows you
to see where the conceptual electron beam is on the screen. It is especially
useful when coupled with the alternative (or debugging) colours mode.

The ALT ON and ALT OFF arguments toggle the alternative colours mode. These
colours allow you to see which pixels on the screen where generated by the
different video components (the playfield, the player sprites, etc.). This mode
displays pixels irrespective of the current HBLANK and VBLANK state. This is
extremely useful when the display has been unmasked with DEBUG OFF.

The SCALE argument takes a floating point number and adjusts the size of the
display.

The OVERLAY ON and OVERLAY OFF arguments toggle a debugging overlay. This
overlay decorates the display with markers showing when during the drawing
process key video events were triggered.`,

	// peripherals (components that might not be present)
	cmdPlusROM: `Controls the attached PlusROM. HOST and PATH can be changed on a per cartridge
basis and only for the duration of the session (the ROM must be changed for it to be permanent).

The NICK and ID can be changed for the session but also be saved to disk and be used across
all PlusROM cartridges.`,

	// user input
	cmdController: `Change the current controller type for the specified player. The AUTO
controller handles changes of controller according to user input and where possible what
can be inferred from the ROM.`,

	cmdPanel: "Inspect and set front panel settings. Switches can be set or toggled.",

	cmdStick: `Set joystick input for Player 0 or Player 1 for the next and
subsequent video cycles.

If the current controller for that player is not a stick (or the auto controller type) then
an error will be returned.

Specify the player with the 0 or 1 arguments.

Note that it is possible to set the stick combinations that would normally not
be possible with a joystick. For example, LEFT and RIGHT set at the same time.`,

	cmdKeyboard: `Set the keyboard input for Player 0 or Player 1 for the next and subsequent
video cycles.

If the current controller for that player is not a keyboard (or the auto controller type) then
an error will be returned.

Specify the player with the 0 or 1 arguments.`,

	// halt conditions
	cmdBreak: `Halt execution of the emulation when a specific value is "loaded" into a named
target. A target is a part of the emulation hardware that can be interegated
about its state. Current targets are:

	the CPU registers (PC, A, X, Y and SP)
	the TV state (FRAMENUM, SCANLINE, HORIZPOS)
	cartidge BANK
	CPU result (RESULT MNEMONIC, RESULT EFFECT, RESULT PAGEFAULT, RESULT BUG)

Specifying an address without a target will be assumed to be break on the PC
and the current cartridge bank. So:

	BREAK <address>

Becomes:

	BREAK PC <address> & BANK <current bank>

A break can depend on the condition of more than one target. Specify complex
conditions with the & operative. For example:

	BREAK SL 10 & X 255

This break will halt execution when, and only when, the TV reports being on the
10th scanline and the X register contains 255. In this instance, the SCANLINE
target has been specified with an abbreviation. Acceptable abbreviations are:

	FRAMENUM -> FRAME, FR
	SCANLINE -> SL
	HORIZPOS -> HP

Resuming execution after a halt will suppress all currently matching breaks
until the conditions change and then match again. In the above example,
execution breaks on SL 10 & X 255. After resumption, the break will not apply
until X changes from 255 to something else and then back again, or SL is hit on
the next frame and X again (or still) has a value of 255.i

Existing breakpoints can be reviewed with the LIST command and deleted with the
DROP or CLEAR commands`,

	cmdTrap: `Cause emulator to halt when specified machine component is touched and changed
to any other value. Traps are very similar to breakpoints in some ways.  They
can be applied to the same set of targets as BREAK (see help for BREAK command
for details).

Existing traps can be reviewed with the LIST command and deleted with the
DROP or CLEAR commands`,

	cmdWatch: `Watch a memory address for activity. Emulation will halt when the watch
is triggered. An individual watch can wait for either read access or write
access of specific address address. Addresses can be specified numerically or
by symbol.

By default, watching a numeric address will specifically watch for read
events. This can be changed by specifying WRITE as the first argument. For
example:
 
	WATCH 0x80

	WATCH WRITE 0x81

The first example watches address 0x80 for read access, while the second will
watch for write access of address 0x81. To watch a single address for both read and
write access, two watches are required.

Symbolic addresses can be used although you must be mindful of whether the symbol refers to a
read or write address. For example:

	WATCH VSYNC

is not allowed because VSYNC is a read-only symbol and we default to specifying
read watches. You must instead specify it as a write watch:

	WATCH WRITE VSYNC

A watch can also watch for a specific value to be written or read from the specified
address.

	WATCH WRITE 0x80 10

The above example will watch for the value 10 (decimal) to be written to memory
address 0x80.

The optional ANY (or MIRRORS) argument will instruct the WATCH to match on any address which is
the equivalent (or mirror) or the one specified. For example:

	WATCH ANY 0xf000

Will watch for a read access of the cartridge address 0xf000 or any of it's mirrors. In this instance,
if the CPU attempts to read 0x1000, the watch will match.

Existing watches can be reviewed with the LIST command and deleted with the
DROP or CLEAR commands`,

	cmdTrace: `"Trace activity on the specied memory address. This means any activity, read or write.
There is no way to isolate one activity from the other. Use WATCH for that.

When activity at a TRACED address is found the terminal will display the CPU instruction that caused
the activity. For example, as a result of "TRACE 0x80":

	$f006 STA VSYNC,X -> write 0x0080 (RAM)
	$faa3 STA $80 -> write 0x0080 (RAM)
	$f0ed LDA $80 -> read 0x0080 (RAM)

There is currently no ONTRACE command.

Generally, WATCH is a more flexible instrument but TRACE can be useful to quickly gather information
about an address.`,

	cmdList:  "List currently defined BREAKS, TRAPS, WATCHES and TRACES.",
	cmdDrop:  "Drop a specific BREAK, TRAP, WATCH or TRACE condition, using the number of the condition reported by LIST.",
	cmdClear: "Clear all BREAKS, TRAPS, WATCHES and TRACES.",

	// meta
	cmdPrefs: "Set preferences for debugger.",
	cmdLog: `Print log to terminal. The LAST argument will cause the most recent log entry to be printed.

Note that while "ONSTEP LOG LAST" is a valid construct it may not print what you expect - it will always print the last
log entry after every step, even if the last log entry is not new. "ONSTEP LOG LAST; LOG CLEAR" is maybe more intuitive
but with the maybe unwanted side effect of clearing the log.`,
}
