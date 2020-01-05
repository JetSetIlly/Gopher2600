# Gopher2600

Gopher 2600 is a more-or-less complete emulation of the Atari VCS. It is
written in Go and was begun as a project for learning that language. It has
minimal dependencies.

The files presented herewith are for the emulator only, you will have to
provide your own Atari VCS ROMs. In the future I plan to bundle and
distribute all the test ROMs that I have used during development, along with
regression databases.

The following document is an outline of the project only. Further documentation
can be viewed with the Go documentation system.  With godoc installed run the
following in the project directory:

> GOMOD=$(pwd) godoc -http=localhost:1234 -index >/dev/null &

## Project Features

* Debugger
	* CPU and Video stepping
	* Breakpoints, traps, watches
	* Script recording and playback
* ROM patching
* Regression database
	* useful for ensuring continuing code accuracy when changing the emulation code
* Setup preferences for individual ROMs
	* Setting of panel switches
	* Auto Application of ROM patches
* Gameplay session recording and playback

## Performance

On a 3GHz i3 processor, the emulator (with SDL display) can reach 60fps or
thereabouts. 

## Resources used

The Stella project (https://stella-emu.github.io/) was used as a reference for
video output. I made the decision not to use or even to look at any of Stella's
implementation details. The exception to this was a peek at the audio
sub-system. Primarily however, Gopher2600's audio implementation references Ron
Fries' original TIASound.c file.

Many notes and clues from the AtariAge message boards. Most significantly the
following threads proved very useful indeed:

* "Cosmic Ark Star Field Revisited"
* "Properly model NUSIZ during player decode and draw"
* "Requesting help in improving TIA emulation in Stella" 
* "3F Bankswitching"

And from and old mailing list:

* "Games that do bad things to HMOVE..." https://www.biglist.com/lists/stella/archives/199804/msg00198.html

These mailing lists and forums have supplied me with many useful test ROMs. I
will package these up and distribute them sometime in the future (assuming I
can get the required permissions).

Extensive references have been made to Andrew Towers' "Atari 2600 TIA Hardware
Notes v1.0"

Cartridge format information was found in Kevin Horton's "Cart Information
v6.0" file (sometimes named bankswitch_sizes.txt)

The "Stella Programmer's Guide" by Steve Wright is of course a key document,
used frequently throughout development.

The 6507 information was taken from Leventhal's "6502 Assembly Language
Programming" and the text file "64doc.txt" v1.0, by John West and Marko Makela.

## ROMs used during development

The following ROMs were used throughout development and compared with the
Stella emulator for accuracy. As far as I can tell the following ROMs work more
or less as you would expect:

### Commercial
* Pitfall
* Adventure
* Barnstormer
* Krull
* He-Man
* ET
* Fatal Run
* Cosmic ark
* Keystone Kapers
* River Raiders
* Tennis
* Wabbit
* Yar's Revenge
* Midnight Madness

### Homebrew
* Thrust (v1.2)
* Hack'em (pac man clone)
* Donkey Kong (v1.0)

### Demos
* tricade by trilobit

## Compilation

The project has most recently been tested with Go v1.13.4. It will not work with
versions earlier than v1.13 because of language features added in that version
(hex and binary literals).

The project uses the Go module system and dependencies will be resolved
automatically.

Compile with GNU Make

> make release

During development, programmers may find it more useful to use the go command
directly

> go run gopher2600.go

## Basic usage

Once compiled run the executable with the help flag:

> ./gopher2600 -help

This will list the available sub-modes. USe the -help flag to get information
about a sub-mode. For example:

> ./gopher2600 debug -help

To run a cartridge, you don't need to specify a sub-mode. For example:

> ./gopher2600 roms/Pitfall.bin

## Debugger

To run the debugger use the DEBUG submode

> ./gopher2600 debug roms/Pitfall.bin

For further help on the debugger, use the HELP command at the terminal.

## Player input

Currently, only joystick controllers are supported and only for player 0.
Moreover, you have to use the keyboard.

### Joystick Player 0

* Cursor keys for stick direction
* Space bar for fire

### Panel

* F1 Panel Select
* F2 Panel Reset
* F3 Color Toggle
* F4 Player 0 Pro Toggle
* F5 Player 0 Pro Toggle

### Debugger

The following keys are only available in the debugger and when the SDL window
is active.

* \` (backtick) Toggle screen masking
* 1 Toggle debugging colors
* 2 Toggle debugging overlay
* \+ Increase screen size
* \- Decrease screen size

All controller/panel functionality is achievable with debugger commands (useful
for scripting).

## Configuration folder

Gopher2600 will look for certain files in a configuration directory. 

> .gopher2600

The UNIX method for hiding files has been used - I have no idea how this works
on Windows etc.

If that directory can be found in the current working directory then that is
the path that will be used. If it can't be found then the user's configuration
folder is checked. On modern Linux based systems, this will be:

> .config/gopher2600

## WASM / HTML5 Canvas

To compile and serve a WASM version of the emulator (no debugger) use:

> make web

The server will be listening on port 2600. Note that you need a file in the
web2600/www folder named "example.bin" for anything to work.

Warning that this is a proof of concept only. The performance is currently
very poor.

## Missing Features

1. Paddle, keyboard, driving, lightgun controllers
1. Not all CPU instructions are implemented. Although adding the missng opcodes
	when encountered should be straightforward.
1. Unimplemented cartridge formats
	* F0 Megaboy
	* AR Arcadia
	* X1 chip (as used in Pitfal 2)
1. Disassembly of some cartridge formats is known to be inaccurate
1. FUTURE and todo.txt files list other known issues


