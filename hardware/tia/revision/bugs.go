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

package revision

type Bug int

const (
	// Late VDEL gfx: The setting of player graphics VDEL data and in the case
	// of GRP1, the ball enable delayed bit. It does not affect the writing of
	// the primary graphics data.
	//
	// For clarity the naming of these preferences refers to the register that
	// is being written. In other words, when GRP0 has been written to the
	// effect of the LateGRP0 option is seen in player 1 delayed gfx data.
	//
	// Example ROM: HeMan
	LateVDELGRP0 Bug = iota
	LateVDELGRP1

	// Late HMOVE Ripple
	//
	// "As with the previous tests, variations in positioning only
	// happen if a strobe to RESxx coincide with an extra motion CLK
	// pulse after an HMOVE. In all the other cases, all the consoles
	// behave the same."
	//
	// https://github.com/stella-emu/stella/issues/699#issuecomment-698004074
	//
	// Example ROM: 36 char demos (36_Char_Interlaced_RESP0_cycle0)
	LateRippleStart

	// Late HMOVE End
	//
	// https://atariage.com/forums/topic/311795-576-and-1008-characters/?tab=comments#comment-4646705
	//
	// Example ROM: 36 char demos (36_Char_Interlaced_RESP0_cycle3)
	LateRippleEnd

	// Late PFx: The setting of the playfield bits happens a video cycle later
	// that it should.
	//
	// Example ROM: Pesco
	LatePFx

	// Late COLUPF: Updating of playfield color register happens a video cycle
	// later than it should.
	//
	// Example ROM: QuickStep
	//
	// I am unsure if this applies to all color registers or just the
	// Playfield. For now, I'm assuming it is only the playfield color
	// register.
	//
	// This is implemented by delaying the servicing of the color register
	// until after the pixel color is selected.
	//
	// Some TIAs that are on the edge of tolerance can also exhibit this
	// behaviour when the TIA is embedded in another device, such as an RGB
	// mod. Explanation of how this can happen:
	//
	// https://atariage.com/forums/topic/307533-atari-rgb-light-sixer-repair/?do=findComment&comment=4559618
	LateCOLUPF

	// In some TIA variations, a HMOVE clock during the non-HBLANK period will
	// cause the regular tick signal to phase out when the sprites HMOVE
	// required flag is set.
	//
	// Example ROMs: Cosmic Ark (starfield) and the barber pole test ROM (barber.a26)
	//
	// An image of the effect on Cosmic Ark can be seen here:
	// http://www.ataricompendium.com/faq/vcs_tia/vcs_tia_cosmic_ark_2.jpg
	LostMOTCK

	// RESPx on HBLANK threshold: the delay when resetting player position is
	// affected by the state of HBLANK. some TIA revisions seem to react even
	// later to HBLANK being reset.
	//
	// This phenomenon seems to be affected by operating temperature. the
	// HeatThreshold() function provides a rudimentary emulation of this.
	//
	// Example ROM: 2/3 sprite demo (labelled bin00004.bin)
	//
	// https://www.biglist.com/lists/stella/archives/199901/msg00089.html
	RESPxHBLANK
)

func (bug Bug) Description() string {
	switch bug {
	case LateVDELGRP0:
		return "GRP1 VDEL gfx on write to GRP0 is not immediate"
	case LateVDELGRP1:
		return "GRP0 VDEL gfx on write to GRP1 is not immediate"
	case LateRippleStart:
		return "HMOVE ripple starts late"
	case LateRippleEnd:
		return "HMOVE ripple ends lat"
	case LatePFx:
		return "PFx bits set late"
	case LateCOLUPF:
		return "COLUPF set late"
	case LostMOTCK:
		return "MOTCK is sometimes ineffective when HBLANK is off"
	case RESPxHBLANK:
		return "RESPx reacts late to HBLANK reset (temperature dependent)"
	}
	return "unknown bug"
}

func (bug Bug) NotableROM() string {
	switch bug {
	case LateVDELGRP0:
		return "He-Man"
	case LateVDELGRP1:
		return "He-Man"
	case LateRippleStart:
		return "36 Character Demos"
	case LateRippleEnd:
		return "36 Character Demos"
	case LatePFx:
		return "Pesco"
	case LateCOLUPF:
		return "Quickstep (can be triggered by RGB Mods)"
	case LostMOTCK:
		return "Cosmic Ark (missile sprite)"
	case RESPxHBLANK:
		return "'2 or 3' sprite demo"
	}
	return "unknown bug"
}
