<img align="left" src="https://github.com/JetSetIlly/Gopher2600-Dev-Docs/blob/master/gopher2600_logo/logo4.png" width="250" alt="gopher2600 logo"/>

# Gopher2600

Gopher2600 is an emulator for the Atari 2600 written in the Go language.

The accuracy of the emulation is very high and there are no known outstanding
problems with the 6507, TIA and RIOT chip emulation. Emulation of the ARM chip
is currently limited to the Thumb subset of instructions but it does include
accurate cycle counting and performance monitoring.

It is an ongoing project and bug reports are welcome.
<br clear="left">

The key features of the emulator:

* [Support for many of the known cartridge formats](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Supported-Cartridge-Mappers) including the [Supercharger](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Supercharger)
* Emulation of the [ARM7TDMI](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/ATM7TDMI-Emulation) as found in the `Harmony` cartridge
* Network access through [PlusROM](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/PlusROM) emulation
* [AtariVox and SaveKey](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/AtariVox-and-SaveKey) support
* CRT Effects
* Three [screenshot](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Creating-Screenshots) methods
* Accurate audio reproduction (and optional stereo output)
* Support for common [TIA revisions](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/TIA-Revisions)
* Implementation of [Movie Cart](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Movie-Cart)
* [Gameplay rewinding](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Rewinding)
* Tracker/Piano Keys visualisation
* [Gameplay recording and playback](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Recording-Gamplay)
* Support for (and auto-detection of) the stick, paddle, keypad and also Sega Genesis style [controllers](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Hand-Controllers-and-Front-Panel)

The graphical [debugger](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Debugger):

* Optional Color Clock level interaction
* Breakpoints, traps, watches on various CPU, TIA and RIOT targets
* Specialist windows for specific cartridge types (eg. supercharger tape)
* Terminal interface (headless operation optional)
* Script recording and playback
* ARM peformance monitoring

Logo is based on [Gopherize.me](https://github.com/matryer/gopherize.me) which itself is based on the work of [Ashley McNamara](https://github.com/ashleymcnamara/gophers) and is [licensed under the Creative Commons](https://github.com/ashleymcnamara/gophers/blob/master/LICENSE).

## Documentation

User documentation for the emulator can be found in the [Gopher2600-Docs repository](https://github.com/JetSetIlly/Gopher2600-Docs/) and in particular the [live wiki](https://github.com/JetSetIlly/Gopher2600-Docs/wiki).

Development & Maintenance documentation can be found in the [Gopher2600-Dev-Docs repository](https://github.com/JetSetIlly/Gopher2600-Dev-Docs/). Also, source level documentation (for the most recent release) can be found on [go.dev](https://pkg.go.dev/github.com/jetsetilly/gopher2600).

## Example Screenshots

The following [screenshots](https://github.com/JetSetIlly/Gopher2600-Docs/wiki/Creating-Screenshots) were taken in `playmode` with CRT effects enabled.

<table align="center">
	<tr>
		<td align="center">
			<img src="https://github.com/JetSetIlly/Gopher2600-Docs/blob/master/screenshots/pitfall.jpg" height="150" alt="pitfall"/>
		</td>
		<td align="center">
			<img src="https://github.com/JetSetIlly/Gopher2600-Docs/blob/master/screenshots/chiphead.jpg" height="150" alt="chiphead"/>
		</td>
	</tr>
	<tr>
		<td align="center">
			<img src="https://github.com/JetSetIlly/Gopher2600-Docs/blob/master/screenshots/thrust.jpg" height="150" alt="thrust"/>
		</td>
		<td align="center">
			<img src="https://github.com/JetSetIlly/Gopher2600-Docs/blob/master/screenshots/mangoesdown.jpg" height="150" alt="man goes down"/>
		</td>
	</tr>
	<tr>
		<td align="center">
			<img src="https://github.com/JetSetIlly/Gopher2600-Docs/blob/master/screenshots/beast.jpg" height="150" alt="soul of the beast"/>
		</td>
		<td align="center">
			<img src="https://github.com/JetSetIlly/Gopher2600-Docs/blob/master/screenshots/genesis.jpg" height="150" alt="egypt genesis"/>
		</td>
	</tr>
	<tr>
		<td align="center">
			<img src="https://github.com/JetSetIlly/Gopher2600-Docs/blob/master/screenshots/draconian.jpg" height="150" alt="draconian"/>
		</td>
		<td align="center">
			<img src="https://github.com/JetSetIlly/Gopher2600-Docs/blob/master/screenshots/zookeeper.jpg" height="150" alt="zookeeper"/>
		</td>
	</tr>
</table>

ROMs shown: Pitfall; [Chiphead](https://www.pouet.net/prod.php?which=68505); Thrust; Man Goes Down; [Soul of the Beast](https://aeriform.itch.io/beast); [Egypt 2600BC](https://www.pouet.net/prod.php?which=72716) ; Draconian; [Zookeeper](https://champ.games/downloads)

## Resources used

The Stella project (https://stella-emu.github.io/) was used as a visual
reference for video output. In the absence of VCS hardware (which I didn't have
during initial TIA development) Stella was a good alternative for checking
the accuracy of video output.

In addition, Stella was used as reference in the following areas:

* During the development of the CDF cartridge formats. These recent formats don't
seem to be documented anywhere accept in the Stella source.

* Cartridge fingerprints for ParkerBros, Wickstead Design and SCABS.

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

The "Stella Programmer's Guide" by Steve Wright is of course a key document,
used frequently throughout development.

Colour values for NTSC and PAL television signals taken from https://www.qotile.net/minidig/docs/tia_color.html

The TIA Audio implementation is based almost entirely on the work of Chris Brenner.

https://atariage.com/forums/topic/249865-tia-sounding-off-in-the-digital-domain/

Musical information as seen in the tracker window taken from Random Terrain.

https://www.randomterrain.com/atari-2600-memories-music-and-sound.html

The 6507 information was taken from Leventhal's "6502 Assembly Language
Programming" and the text file "64doc.txt" v1.0, by John West and Marko Makela.

Undocumented 650x instructions and implementation details in "6502/6510/8500/8502 Opcode matrix" 

http://www.oxyron.de/html/opcodes02.html

6502 functional tests from https://github.com/Klaus2m5/6502_65C02_functional_tests

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
taken from the DWARF2 and DWARF4 standards

https://dwarfstd.org/doc/dwarf-2.0.0.pdf

https://dwarfstd.org/doc/DWARF4.pdf

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

At various times during the development of this project, the following people
have provided advice and encouragement: Andrew Rice, David Kelly. And those
from AtariAge who have provided testing, advice and most importantly,
encouragement (alphabetically): alex_79; Al Nafuur; Andrew Davie; DirtyHairy;
John Champeau; MarcoJ; MrSQL; Rob Bairos; Spiceware; Thomas Jenztsch; Zachary
Scolaro; ZeroPageHomebrew
