# Gopher2600

<img align="right" src="https://github.com/JetSetIlly/Gopher2600-Docs/blob/master/screenshots/new_pitfall_as_banner.png" width=50% />

Gopher2600 is an emulator for the Atari 2600. Accuracy is very high and and there are no known problems with the emulation of the 6507, TIA or RIOT chips.

The emulator is suitable for both playing 2600 games and for developing new games. In particular, the debugging features available for developers of CDFJ, DPC+ and ELF type cartridges (cartridges that make use of an ARM coprocessor) are unique.

Most ["bank switching"](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Supported-Cartridge-Mappers) types are supported, including the above mentioned CDFJ, DPC+ and ELF types. Also notable is the support for [Supercharger](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Supercharger) and in particular, the loading of Supercharger tapes stored as a WAV or MP3 file. Also supported is the [Movie Cart](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Movie-Cart)

Games using [PlusROM](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/PlusROM) features are also supported.

The [AtariVox and SaveKey](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/AtariVox-and-SaveKey) peripherals are emulated. Uniquely, the AtariVox voice features can be emulated with the third-party application `Festival`. Although the voice reproduction is imperfect, it is helpful to hear feedback for AtariVox voice commands. A subtitler for the AtariVox voice output is also available. Again, this is imperfect and only shows phonetic spellings.

The vintage CRT television, contemporaneous with the console, is emulated with a selection of interference effects and bevel images. This is an ongoing area of development and will be improved further in the future.

Flexible [screenshot](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Creating-Screenshots) features so that games using "flicker" kernels can be effectively captured.

Accurate audio reproduction with optional stereo output. The intensity of the stereo effect can be altered.

Over the course of its lifetime, the Atari2600 came in several versions which introduced subtle variations in the TIA chip. Gopher2600 supports the common [TIA revisions](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/TIA-Revisions)

For the player who wants to master a game quickly there is convenient [gameplay rewinding](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Rewinding). The rewinding feature is also available in the emulator's debugger in the form of a "timeline" window.

Gameplay can also be [recorded and played back](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Recording-Gamplay). In addition to keeping gameplay sessions for future study or enjoyment, recordings are also useful for testing purposes during game development.

The standard controllers are supported. The joystick, the paddle, the keypad and also Sega Genesis style [controllers](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Hand-Controllers-and-Front-Panel).

The graphical [debugger](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Debugger) provides a powerful interface to the internals of the Atari 2600. The state of the console can be inspected and rewound at the CPU and Colour Clock level. Breakpoints and watches can be set on many areas of the CPU, TIA and RIOT and the same areas can be be peeked and poked as required.

In addition to the graphical windows of the debugger, a terminal interface is provided.

The most powerful use of the debugger however is the debugging and profiling of cartridges that use an ARM coprocessor (Harmony, PlusCart, etc.) When the game to be debugged is [compiled correctly](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Preparing-ARM-Enabled-Projects-for-the-Gopher2600-Debugger) Gopher2600 can use the DWARF information to  profile performance; to identify incorrect code flow; and to identify incorrect use of the program's global and local variables.

## Basic Usage

Launching the emulator from the desktop or command line will open the emulator in `playmode`. The ROM selector allows you to select a 2600 ROM file from your collection. The ROM selector can be opened at any time by pressing the `TAB` key.

For joystick games, player one can be controlled with the keyboard - using cursor keys and the space bar for fire. Alternatively, any attached gamepad can be used. For games that do not use a joystick see the [controllers wiki entry](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Hand-Controllers-and-Front-Panel) for more details. This page also lists the keys for the console's panel switches.

The preferences window can be opened by pressing the `F10` key. Amongst other settings, the television can be adjusted from this window and in particular, the CRT effects can be changed or disabled if necessary.

Finally, the debugger can be activated by pressing the backquote `` ` `` key. 

All other documentation and help for Gopher2600 is listed in the following section.

## Documentation and Help

User documentation for the emulator can be found in the project [wiki](https://github.com/JetSetIlly/Gopher2600-Docs/wiki).

Development & Maintenance documentation can be found in the [Gopher2600-Dev-Docs repository](https://github.com/JetSetIlly/Gopher2600-Dev-Docs/).

Source level documentation can be found on the [Go package documentation site](https://pkg.go.dev/github.com/jetsetilly/gopher2600).

There is also a project [blog](https://jetsetilly.github.io/Gopher2600-Blog/) which will be updated periodically with topical articles. 

## Resources used

The Stella project (https://stella-emu.github.io/) was used as a visual
reference for video output. In the absence of VCS hardware (which I didn't have
during initial TIA development) Stella was a good alternative for checking
the accuracy of video output.

In addition, Stella was used as reference in the following areas:

* During the development of the CDF cartridge formats. These recent formats don't
seem to be documented anywhere accept in the Stella source.

* Cartridge fingerprints for ParkerBros, Wickstead Design, SCABS, UA and JANE.

* As a reference for the audio implementation (the 6502.ts project was also
  referenced for this reason).

* Fingerprint patterns for automated controller/peripheral selection.

In all these instances, primary sources of information could not be found.

(Earlier revision of Gopher2600 used Stella a reference for the EF cartridge
format. However, the implementation has been vastly simplified by declaring EF
to be a nothing more than a 64k Atari ROM. The discussion that led to this
revelation can be found in the link below.)

https://forums.atariage.com/topic/346341-64ksc-multi-sprite-roms-error-out-in-stella-for-me/page/2/#comment-5188396

Many notes and clues from the AtariAge message boards. Most significantly the
following threads proved very useful indeed:

* "Cosmic Ark Star Field Revisited"
* "Properly model NUSIZ during player decode and draw"
* "Requesting help in improving TIA emulation in Stella" 
* "3F Bankswitching"
* "TIA Sounding Off in the Digital Domain"

And from and old mailing list:

* "Games that do bad things to HMOVE..." https://www.biglist.com/lists/stella/archives/199804/msg00198.html

These mailing lists and forums have supplied me with many useful test ROMs. I
aim to package these up and distribute them sometime in the future (assuming I
can get the required permissions).

Extensive references have been made to Andrew Towers' "Atari 2600 TIA Hardware
Notes v1.0"

Cartridge format information was found in Kevin Horton's "Cart Information
v6.0" file (sometimes named bankswitch_sizes.txt)

The WF8 format discussed here on AtariAge

https://forums.atariage.com/topic/367157-smurf-rescue-alternative-rom-with-wf8-bankswitch-format/

The "Stella Programmer's Guide" by Steve Wright is of course a key document,
used frequently throughout development.

Colour value information for NTSC, PAL and SECAM palettes taken from (in versions prior to v0.40.0):

https://www.qotile.net/minidig/docs/tia_color.html
https://www.randomterrain.com/atari-2600-memories-tia-color-charts.html
https://forums.atariage.com/topic/240893-question-for-homebrewers-atari-2600-colors/

Ongoing work to improve palette handling with a new mathematical model is ongoing and was prompted by this thread:

https://forums.atariage.com/topic/375698-how-are-ntsc-console-colors-really-set-up

The TIA Audio implementation is based almost entirely on the work of Chris Brenner.

https://atariage.com/forums/topic/249865-tia-sounding-off-in-the-digital-domain/

Additional work on volume sampling a result of this thread:

https://forums.atariage.com/topic/370460-8-bit-digital-audio-from-2600/

Musical information as seen in the tracker window taken from Random Terrain.

https://www.randomterrain.com/atari-2600-memories-music-and-sound.html

The 6507 information was taken from Leventhal's "6502 Assembly Language
Programming" and the text file "64doc.txt" v1.0, by John West and Marko Makela.

Undocumented 650x instructions and implementation details in "6502/6510/8500/8502 Opcode matrix" 

http://www.oxyron.de/html/opcodes02.html

6502 overflow flag

https://www.righto.com/2012/12/the-6502-overflow-flag-explained.html

Details about the 6502 decimal mode

http://www.6502.org/tutorials/decimal_mode.html

https://forums.atariage.com/topic/163876-flags-on-decimal-mode-on-the-nmos-6502

6502 functional tests from https://github.com/Klaus2m5/6502_65C02_functional_tests and single step tests from https://github.com/SingleStepTests/65x02/tree/main/6502

US Patent Number 4,644,495 was referenced for the implementation of the DPC cartridge format
(the format used in Pitfall 2) https://patents.google.com/patent/US4644495/en

US patent 4,485,457A was used to help implement the CBS cartridge format
https://patents.google.com/patent/US4485457A/en

European patent 84300730.3 was used to help implement the SCABS cartridge format
https://worldwide.espacenet.com/patent/search/family/023848640/publication/EP0116455A2?q=84300730.3

DPC+ format implemented according to notes provided by Spiceware https://atariage.com/forums/topic/163495-harmony-dpc-programming and https://atariage.com/forums/blogs/entry/11811-dpcarm-part-6-dpc-cartridge-layout/

DPC+ARM information on Spiceware's blog https://atariage.com/forums/blogs/entry/11712-dpc-arm-development/?tab=comments#comment-27116

The "Mostly Inclusive Atari 2600 Mapper / Selected Hardware Document" (dated 03/04/12) by Kevin Horton

Supercharger information from the Kevin Horton document above and also the `sctech.txt` document

Information about the SpeakJet chip found in the AtariVox peripheral
https://people.ece.cornell.edu/land/courses/ece4760/Speech/speakjetusermanual.pdf

Reference for the ARM7TDMI-S, as used in the Harmony cartridge formats:

https://developer.arm.com/documentation/ddi0234/b

For detail about the Thumb instruction set the following document was
preferred. Mnemonics used in the ARM disassembly are from this document:

http://bear.ces.cwru.edu/eecs_382/ARM7-TDMI-manual-pt1.pdf

Further information from the ARM Architecture Reference Manual:

http://www.ecs.csun.edu/~smirzaei/docs/ece425/arm7tdmi_instruction_set_reference.pdf

https://www.cs.miami.edu/home/burt/learning/Csc521.141/Documents/arm_arm.pdf

Specific information about UXP ARM7TDMI-S 

https://www.nxp.com/docs/en/user-guide/UM10161.pdf

Thumb-2 information in the ARM Architecture Reference Manual Thumb-2 Supplement

https://documentation-service.arm.com/static/5f1066ca0daa596235e7e90a

and the "ARMv7-M Architecture Reference Manual" can be found at:

https://documentation-service.arm.com/static/606dc36485368c4c2b1bf62f

Specific information about the STM32F407 used in the UnoCart and PlusCart can
be found at:

https://www.st.com/resource/en/reference_manual/dm00031020-stm32f405-415-stm32f407-417-stm32f427-437-and-stm32f429-439-advanced-arm-based-32-bit-mcus-stmicroelectronics.pdf

In relation to ARM development, information about the DWARF format is being
taken from the DWARF2 and DWARF4 standards:

https://dwarfstd.org/doc/dwarf-2.0.0.pdf

https://dwarfstd.org/doc/DWARF4.pdf

ARM specific DWARF information taken from:

https://github.com/ARM-software/abi-aa/releases/download/2023Q1/aadwarf32.pdf

Information about ELF for the ARM Architecture, particular information about relocation types, taken from:

https://prog5.gricad-pages.univ-grenoble-alpes.fr/Projet/IHI0044F_aaelf.pdf

https://kolegite.com/EE_library/standards/ARM_ABI/aaelf32.pdf

## Other Software / Libraries

The following projects are used in the `Gopher2600` project:

* https://github.com/ocornut/imgui
* https://github.com/inkyblackness/imgui-go
* https://github.com/veandco/go-sdl2
* https://github.com/go-gl/gl
* https://github.com/hajimehoshi/go-mp3
* https://github.com/pkg/term
* https://github.com/go-audio/wav
* https://github.com/sahilm/fuzzy

* FontAwesome
	* https://fontawesome.com/
	* licensed under the Font Awesome Free License
* Hack-Regular 
	* https://github.com/source-foundry/Hack
	* licensed under the MIT License
* JetBrainsMono
	* https://github.com/JetBrains/JetBrainsMono
	* licensed under the OFL-1.1 License

Both 6502.ts and Stella were used as reference for the Audio implementation.

Some ideas for the fragment shaders taken from:

* https://github.com/libretro/glsl-shaders/blob/master/crt/shaders/crt-pi.glsl
* https://www.shadertoy.com/view/ltB3zD
* https://github.com/mattiasgustavsson/crtview
* https://gist.github.com/Beefster09/7264303ee4b4b2086f372f1e70e8eddd

The Festival Speech Synthesis System is an optional program that can be run
alongside the emulator for [AtariVox](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/AtariVox-and-SaveKey) support

http://www.festvox.org/docs/manual-2.4.0/festival_toc.html

## Personal Thanks and Acknowledgements

<img align="right" src="https://github.com/JetSetIlly/Gopher2600-Dev-Docs/blob/master/gopher2600_logo/logo4.png" height="400px" />

* Andrew Rice
* David Kelly
* And those from the wider Atari community:
	* alex_79
	* Andrew Davie
	* Christian Speckner (DirtyHairy)
	* Darrell Spice (Spiceware)
   	* Dion Olsthoorn (Dionoid)
	* Fred Quimby (Batari)
	* James O'Brien (ZeroPageHomebrew)
	* John Champeau (Champ Games)
	* Marco Johannes (MarcoJ)
	* MrSQL
	* Rob Bairos
	* Rob Tuccitto (Trebor)
	* Thomas Jenztsch
	* Wolfgang Stubig (Al Nafuur)
	* Zachary Scolaro

_Logo is based on [Gopherize.me](https://github.com/matryer/gopherize.me) which itself is based on the work of [Ashley McNamara](https://github.com/ashleymcnamara/gophers) and is [licensed under the Creative Commons](https://github.com/ashleymcnamara/gophers/blob/master/LICENSE)_
