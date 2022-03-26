<img align="left" src=".resources/logo.png" width="250" alt="gopher2600 logo"/>

# Gopher2600

Gopher2600 is an emulator for the Atari 2600 written in the Go language.

The accuracy of the emulation is very high and there are no known outstanding problems with the 6507, TIA and RIOT chip emulation.

It is an ongoing project and bug reports are welcome.
<br clear="left">

The key features of the emulator:

* [Support for many of the known cartridge formats](https://github.com/JetSetIlly/Gopher2600/wiki/Supported-Cartridge-Mappers)
* Emulation of the [ARM7TDMI](https://github.com/JetSetIlly/Gopher2600/wiki/ATM7TDMI-Emulation) as found in the `Harmony` cartridge
* Network access through [PlusROM](https://github.com/JetSetIlly/Gopher2600/wiki/PlusROM) emulation
* [AtariVox and SaveKey](https://github.com/JetSetIlly/Gopher2600/wiki/AtariVox-and-SaveKey) support
* CRT Effects
* Accurate audio reproduction (and optional stereo output)
* Support for common [TIA revisions](https://github.com/JetSetIlly/Gopher2600/wiki/TIA-Revisions)
* Implementation of [Movie Cart](https://github.com/JetSetIlly/Gopher2600/wiki/Movie-Cart)
* [Gameplay rewinding](https://github.com/JetSetIlly/Gopher2600/wiki/Rewinding)
* Tracker/Piano Keys visualisation
* [Gameplay recording and playback](https://github.com/JetSetIlly/Gopher2600/wiki/Recording-Gamplay)
* Support for (and auto-detection of) the stick, paddle, keypad and also Sega Genesis style [controllers](https://github.com/JetSetIlly/Gopher2600/wiki/Hand-Controllers-and-Front-Panel)

The graphical [debugger](https://github.com/JetSetIlly/Gopher2600/wiki/Debugger):

* Optional Color Clock level interaction
* Breakpoints, traps, watches on various CPU, TIA and RIOT targets
* Specialist windows for specific cartridge types (eg. supercharger tape)
* Terminal interface (headless operation optional)
* Script recording and playback

Logo is based on [Gopherize.me](https://github.com/matryer/gopherize.me) which itself is based on the work of [Ashley McNamara](https://github.com/ashleymcnamara/gophers) and is [licensed under the Creative Commons](https://github.com/ashleymcnamara/gophers/blob/master/LICENSE).

## Example Screenshots

The following [screenshots](#screenshots) were taken in playmode with CRT effects enabled.

<table align="center">
	<tr>
		<td align="center">
			<img src=".screenshots/games/pitfall.jpg" height="200" alt="pitfall"/>
		</td>
		<td align="center">
			<img src=".screenshots/games/heman.jpg" height="200" alt="he-man"/>
		</td>
	</tr>
	<tr>
		<td align="center">
			<img src=".screenshots/games/krull.jpg" height="200" alt="krull"/>
		</td>
		<td align="center">
			<img src=".screenshots/games/ladybug.jpg" height="200" alt="ladybug"/>
		</td>
	</tr>
	<tr>
		<td align="center">
			<img src=".screenshots/games/thrust.jpg" height="200" alt="thrust"/>
		</td>
		<td align="center">
			<img src=".screenshots/games/mangoesdown.jpg" height="200" alt="man goes down"/>
		</td>
	</tr>
	<tr>
		<td align="center">
			<img src=".screenshots/games/beast.jpg" height="200" alt="soul of the beast"/>
		</td>
		<td align="center">
			<img src=".screenshots/games/chiphead.jpg" height="200" alt="chiphead"/>
		</td>
	</tr>
	<tr>
		<td align="center">
			<img src=".screenshots/games/genesis.jpg" height="200" alt="egypt genesis"/>
		</td>
		<td align="center">
			<img src=".screenshots/games/draconian.jpg" height="200" alt="draconian"/>
		</td>
	</tr>
	<tr>
		<td align="center">
			<img src=".screenshots/games/galagon.jpg" height="200" alt="galagon"/>
		</td>
		<td align="center">
			<img src=".screenshots/games/turbo.jpg" height="200" alt="turbo"/>
		</td>
	</tr>
	<tr>
		<td align="center">
			<img src=".screenshots/games/zookeeper.jpg" height="200" alt="zookeeper"/>
		</td>
		<td align="center">
			<img src=".screenshots/games/moviecart.jpg" height="200" alt="moviecart"/>
		</td>
	</tr>
</table>

Games shown: Pitfall; He-Man; Krull; Ladybug; Thrust; Man Goes Down; [Soul of the Beast](https://aeriform.itch.io/beast); Chiphead; Egypt 2600BC by Genesis Project; Draconian; [Galagon](https://champ.games/downloads); [Turbo](https://champ.games/downloads); [Zookeeper](https://champ.games/downloads); [Movie Cart](https://github.com/JetSetIlly/Gopher2600/wiki/Movie-Cart).


## Usage

Usage documentation for the emulator can be found in the [wiki pages](https://github.com/JetSetIlly/Gopher2600/wiki).

## Resources used

The Stella project (https://stella-emu.github.io/) was used as a reference for
video output. In the absence of VCS hardware (which I don't have) Stella was
the only option I had for checking video accuracy.

In addition, Stella was used as reference 

* During the development of the CDF cartridge formats. These recent formats don't
seem to be documented anywhere accept in the Stella source.

* ParkerBros fingerprint taken from Stella. I can't remember why I did this but
a comment in the fingerprint.go file says I did.

* As a reference for the audio implementation (the 6502.ts project was also
  referenced for this reason).

* The EF cartridge format.

* Fingerprint patterns for automated controller/peripheral selection.

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
will package these up and distribute them sometime in the future (assuming I
can get the required permissions).

Extensive references have been made to Andrew Towers' "Atari 2600 TIA Hardware
Notes v1.0"

Cartridge format information was found in Kevin Horton's "Cart Information
v6.0" file (sometimes named bankswitch_sizes.txt)

The "Stella Programmer's Guide" by Steve Wright is of course a key document,
used frequently throughout development.

The TIA Audio implementation is based almost entirely on the work of Chris Brenner.

https://atariage.com/forums/topic/249865-tia-sounding-off-in-the-digital-domain/

Musical information as seen in the tracker window taken from Random Terrain.

https://www.randomterrain.com/atari-2600-memories-music-and-sound.html

The 6507 information was taken from Leventhal's "6502 Assembly Language
Programming" and the text file "64doc.txt" v1.0, by John West and Marko Makela.

US Patent Number 4,644,495 was referenced for the implementation of the DPC cartridge format
(the format used in Pitfall 2)

DPC+ format implemented according to notes provided by Spiceware https://atariage.com/forums/topic/163495-harmony-dpc-programming
and

https://atariage.com/forums/blogs/entry/11811-dpcarm-part-6-dpc-cartridge-layout/

DPC+ARM information on Spiceware's blog

https://atariage.com/forums/blogs/entry/11712-dpc-arm-development/?tab=comments#comment-27116

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

## Other Software / Libraries

The following projects are used in the `Gopher2600` project:

* https://github.com/ocornut/imgui
* https://github.com/inkyblackness/imgui-go
* https://github.com/veandco/go-sdl2
* https://github.com/go-gl/gl
* https://github.com/go-audio/audio
* https://github.com/go-audio/wav
* https://github.com/hajimehoshi/go-mp3
* https://github.com/pkg/term

* FontAwesome
	* https://fontawesome.com/
	* licensed under the Font Awesome Free License
* Hack-Regular 
	* https://github.com/source-foundry/Hack
	* licensed under the MIT License
* JetBrainsMono
	* https://github.com/JetBrains/JetBrainsMono
	* licensed under the OFL-1.1 License

Bother 6502.ts and Stella were used as reference for the Audio implementation.

Statsview provided by:

* https://github.com/go-echarts/statsview

For testing instrumentation:

* https://github.com/bradleyjkemp/memviz

Some ideas for the fragment shader taken from:

* https://github.com/libretro/glsl-shaders/blob/master/crt/shaders/crt-pi.glsl
* https://www.shadertoy.com/view/ltB3zD
* https://github.com/mattiasgustavsson/crtview

The Festival Speech Synthsis System is an optional program that can be run
alongside the emulator for [AtariVox](https://github.com/JetSetIlly/Gopher2600/wiki/AtariVox-and-SaveKey) support

http://www.festvox.org/docs/manual-2.4.0/festival_toc.html

## Personal Thanks and Acknowledgements

At various times during the development of this project, the following people
have provided advice and encouragement: Andrew Rice, David Kelly. And those
from AtariAge who have provided testing, advice and most importantly,
encouragement: Al Nafuur; Andrew Davie; Rob Bairos; MrSQL; Thomas Jenztsch;
DirtyHairy; Spiceware; ZeroPageHomebrew; Karl G; alex_79. Thank-you.
