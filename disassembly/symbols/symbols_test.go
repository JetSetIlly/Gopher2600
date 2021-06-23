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

package symbols_test

import (
	"os"
	"testing"

	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/test"
)

func TestDefaultSymbols(t *testing.T) {
	var sym symbols.Symbols

	cart := cartridge.NewCartridge(nil)
	err := sym.ReadSymbolsFile(cart)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}
	tw := &test.Writer{}

	sym.ListSymbols(tw)

	if !tw.Compare(expectedDefaultSymbols) {
		t.Errorf("default symbols list is wrong")
	}
}

func TestFlappySymbols(t *testing.T) {
	var sym symbols.Symbols

	// make a dummy cartridge with the minimum amount of information required
	// for ReadSymbolsFile() to work
	cart := cartridge.NewCartridge(nil)
	cart.Filename = "testdata/flappy.bin"

	err := sym.ReadSymbolsFile(cart)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}

	tw := &test.Writer{}

	sym.ListSymbols(os.Stdout)
	sym.ListSymbols(tw)

	if !tw.Compare(expectedFlappySymbols) {
		t.Errorf("flappy symbols list is wrong")
	}
}

const expectedDefaultSymbols = `Labels
------

Read Symbols
------------
0x0000 -> CXM0P
0x0001 -> CXM1P
0x0002 -> CXP0FB
0x0003 -> CXP1FB
0x0004 -> CXM0FB
0x0005 -> CXM1FB
0x0006 -> CXBLPF
0x0007 -> CXPPMM
0x0008 -> INPT0
0x0009 -> INPT1
0x000a -> INPT2
0x000b -> INPT3
0x000c -> INPT4
0x000d -> INPT5
0x0280 -> SWCHA
0x0281 -> SWACNT
0x0282 -> SWCHB
0x0283 -> SWBCNT
0x0284 -> INTIM
0x0285 -> TIMINT

Write Symbols
-------------
0x0000 -> VSYNC
0x0001 -> VBLANK
0x0002 -> WSYNC
0x0003 -> RSYNC
0x0004 -> NUSIZ0
0x0005 -> NUSIZ1
0x0006 -> COLUP0
0x0007 -> COLUP1
0x0008 -> COLUPF
0x0009 -> COLUBK
0x000a -> CTRLPF
0x000b -> REFP0
0x000c -> REFP1
0x000d -> PF0
0x000e -> PF1
0x000f -> PF2
0x0010 -> RESP0
0x0011 -> RESP1
0x0012 -> RESM0
0x0013 -> RESM1
0x0014 -> RESBL
0x0015 -> AUDC0
0x0016 -> AUDC1
0x0017 -> AUDF0
0x0018 -> AUDF1
0x0019 -> AUDV0
0x001a -> AUDV1
0x001b -> GRP0
0x001c -> GRP1
0x001d -> ENAM0
0x001e -> ENAM1
0x001f -> ENABL
0x0020 -> HMP0
0x0021 -> HMP1
0x0022 -> HMM0
0x0023 -> HMM1
0x0024 -> HMBL
0x0025 -> VDELP0
0x0026 -> VDELP1
0x0027 -> VDELBL
0x0028 -> RESMP0
0x0029 -> RESMP1
0x002a -> HMOVE
0x002b -> HMCLR
0x002c -> CXCLR
0x0280 -> SWCHA
0x0281 -> SWACNT
0x0294 -> TIM1T
0x0295 -> TIM8T
0x0296 -> TIM64T
0x0297 -> T1024T
`

const expectedFlappySymbols = `Labels
------
0x1000 -> DATA_SEGMENT
0x1002 -> EMPTY
0x100a -> TEXT_OK
0x1012 -> TEXT_QMARK
0x101a -> WINGS
0x1022 -> WINGS_FLAT
0x102a -> WINGS_DOWN
0x1032 -> HEADS
0x103a -> HEAD_BOY_A
0x1042 -> HEAD_GIRL_B
0x104a -> HEAD_BOY_B
0x1052 -> HEADS_TABLE
0x1056 -> _SPLASH
0x1058 -> SPLASH
0x1060 -> DIGIT_0
0x1065 -> DIGIT_1
0x106a -> DIGIT_2
0x106f -> DIGIT_3
0x1074 -> DIGIT_4
0x1079 -> DIGIT_5
0x107e -> DIGIT_6
0x1083 -> DIGIT_7
0x1088 -> DIGIT_8
0x108d -> DIGIT_9
0x1092 -> DIGIT_TABLE
0x109c -> FOLIAGE
0x10ba -> FOREST_MID_0_INIT
0x10bb -> FOREST_MID_1_INIT
0x10bc -> FOREST_MID_2_INIT
0x10bd -> FOREST_STATIC_0
0x10be -> FOREST_STATIC_1
0x10bf -> FOREST_STATIC_2
0x1100 -> SET_OBSTACLE_TABLE
0x1137 -> FINE_POS_TABLE
0x1204 -> OBSTACLES
0x120b -> BRANCHES
0x1212 -> EASY_FLIGHT_PATTERN
0x1228 -> __FINE_POS_TABLE
0x1237 -> SFX_TABLE
0x123d -> SFX_FLAP
0x1243 -> SFX_COLLISION
0x1255 -> SFX_SPLASH
0x127f -> setup
0x1285 -> 17.CLEAR_STACK
0x128d -> title_screen
0x128f -> 20.VSLP1
0x1296 -> 18.vblank
0x12a5 -> 23.done
0x12ab -> 24.vblank_loop
0x12b4 -> 18.visible_loop
0x12b9 -> 18.overscan
0x12c1 -> 25.overscan_loop
0x12c9 -> 18.end_title_screen
0x12d2 -> 27.overscan_kernel
0x12da -> 27.overscan_loop
0x12df -> game_state_init
0x1332 -> game_restart
0x133c -> 32.done
0x134b -> 31.done_hiscore
0x1350 -> 34.VSLP1
0x135e -> 36.coarse_div
0x1362 -> 36.done_coarse_div
0x1375 -> 38.coarse_div
0x1379 -> 38.done_coarse_div
0x13bb -> game_vsync
0x13bd -> 43.VSLP1
0x13c4 -> game_vblank
0x13de -> 44.far_jmp_collision
0x13e1 -> 44.far_jmp_drown
0x13e4 -> 44.far_jmp_approach
0x13e7 -> 44.far_jmp_play
0x13ea -> game_vblank_ready
0x13f2 -> 48.done
0x1408 -> 49.no_store_index
0x141a -> 47.ready_state_triage
0x1428 -> 47.update_foliage
0x1431 -> 51.rotate_forest
0x143d -> 51.jump_tree
0x1444 -> 51.cont_forest
0x144d -> 51.carry_tree
0x1453 -> 51.forest_done
0x1455 -> 47.prepare_display
0x145b -> 47.display_empty
0x1466 -> 47.display_ready_logo
0x147d -> 54.coarse_div
0x1481 -> 54.done_coarse_div
0x1492 -> 56.coarse_div
0x1496 -> 56.done_coarse_div
0x14a5 -> game_vblank_death_collision
0x14b2 -> 57.update_foliage
0x14bb -> 59.foliage_updated
0x14c0 -> 57.update_bird
0x14ca -> 57.use_wings_up
0x14d1 -> 57.use_wings_flat
0x14d8 -> 57.use_wings_down
0x14dc -> 57.wings_updated
0x14ed -> 57.update_pattern_idx
0x14f9 -> 61.store_index
0x1505 -> 57.enter_drowning_state
0x1525 -> 57.prepare_display
0x152a -> 66.coarse_div
0x152e -> 66.done_coarse_div
0x1542 -> 67.coarse_div
0x1546 -> 67.done_coarse_div
0x1555 -> game_vblank_death_drown
0x1562 -> 68.update_foliage
0x156b -> 70.foliage_updated
0x1570 -> 68.update_bird
0x157b -> 68.drowning_end
0x157e -> 68.prepare_display
0x1583 -> 73.coarse_div
0x1587 -> 73.done_coarse_div
0x159b -> 74.coarse_div
0x159f -> 74.done_coarse_div
0x15b0 -> 76.coarse_div
0x15b4 -> 76.done_coarse_div
0x15cb -> 79.coarse_div
0x15cf -> 79.done_coarse_div
0x15de -> 68.show_obstacle_1
0x15e3 -> 81.coarse_div
0x15e7 -> 81.done_coarse_div
0x15f3 -> 68.flipped_obstacles
0x160a -> game_vblank_approach
0x161e -> 84.hpos_done
0x162c -> 84.done_completion_test
0x1639 -> 86.far_jmp_sprite
0x163c -> game_vblank_foliage
0x1645 -> 88.rotate_forest
0x1651 -> 88.jump_tree
0x1658 -> 88.cont_forest
0x1661 -> 88.carry_tree
0x1667 -> 88.forest_done
0x166c -> game_vblank_collisions
0x1689 -> 89.reset_obstacle_0
0x1696 -> 89.reset_obstacle_1
0x16aa -> 89.bird_collision
0x16b8 -> 92.no_store_index
0x16c4 -> 89.done_vblank_collisions
0x16c7 -> game_vblank_sprite
0x16cf -> 95.done
0x16ec -> 94.flip_sprite_use_flat
0x16f0 -> 94.flip_sprite_end
0x1700 -> 94.use_wings_up_sprite
0x1707 -> 94.use_glide_sprite
0x170b -> 94.sprite_set
0x1716 -> 94.begin_drowning
0x1739 -> 94.limit_height
0x173b -> 94.update_pattern_idx
0x1747 -> 100.store_index
0x1749 -> 94.fly_end
0x174e -> 104.coarse_div
0x1752 -> 104.done_coarse_div
0x1766 -> 105.coarse_div
0x176a -> 105.done_coarse_div
0x177f -> 106.fine_move_done
0x178a -> 107.fine_move_done
0x1791 -> 109.coarse_div
0x1795 -> 109.done_coarse_div
0x17a7 -> 111.coarse_div
0x17ab -> 111.done_coarse_div
0x17b7 -> 101.scoring_check
0x17c8 -> 101.score_obstacle
0x17d4 -> 101.end_scoring
0x17ec -> 113.vblank_loop
0x17f7 -> 114.next_foliage
0x180e -> 114.set_trunk
0x1814 -> 114.new_foliage
0x1816 -> 114.cont_foliage
0x1823 -> game_play_area_prepare
0x183c -> 115.precalc_forest_static
0x1851 -> 115.end_forest_precalc
0x186d -> game_play_area
0x187d -> 117.set_missile_sprites
0x188d -> 117.precalc_missile_size
0x1895 -> 117.done_precalc_missile_size
0x1897 -> 117.precalc_missile_sprites
0x18a2 -> 117.set_player_sprites
0x18ae -> 117.precalc_players_sprites
0x18c3 -> 117.done_precalc_players
0x18cb -> 117.next_scanline
0x18d6 -> 118.draw_swamp
0x1901 -> display_score
0x1923 -> 119.prep_score
0x197e -> 119.prep_hiscore
0x19cc -> 119.tens_digits
0x19da -> 119.scoring_loop
0x19e4 -> 119.next_scanline
0x19ea -> game_overscan
0x1a04 -> 136.done
0x1a0f -> 133.swap_heads
0x1a16 -> 133.done_head_check
0x1a38 -> 133.set_width_for_ready
0x1a40 -> 133.done_set_width
0x1a4a -> 133.next_obstacle
0x1a54 -> 133.next_branch
0x1a7c -> 133.done_drowning_compensation
0x1a82 -> 141.sfx_new_event
0x1a8d -> 141.sfx_queue_event
0x1ac9 -> 141.sfx_cont
0x1ad7 -> 141.sfx_done
0x1adb -> 143.is_positive
0x1ae5 -> 143.positive_reset
0x1aea -> 143.is_negative
0x1af1 -> 143.negative_reset
0x1af3 -> 143.store
0x1af5 -> 144.overscan_loop
0x1afd -> initialisation

Read Symbols
------------
0x0000 -> CXM0P
0x0001 -> CXM1P
0x0002 -> CXP0FB
0x0003 -> CXP1FB
0x0004 -> CXM0FB
0x0005 -> CXM1FB
0x0006 -> CXBLPF
0x0007 -> CXPPMM
0x0008 -> INPT0
0x0009 -> INPT1
0x000a -> INPT2
0x000b -> INPT3
0x000c -> INPT4
0x000d -> INPT5
0x000e -> PF1
0x000f -> PF2
0x0080 -> __MULTI_COUNT_STATE
0x0081 -> __STATE_INPT4
0x0082 -> __STATE_SWCHB
0x0083 -> __SFX_NEW_EVENT
0x0084 -> __SFX_QUEUE_EVENT
0x0085 -> __SFX_SUB_FRAMES
0x0086 -> _localA
0x0087 -> _localB
0x0088 -> _localC
0x0089 -> _localD
0x008a -> _localE
0x008b -> _localF
0x008c -> _localG
0x008d -> PLAY_STATE
0x008e -> SELECTED_HEAD
0x008f -> FLIGHT_PATTERN
0x0091 -> ADDRESS_SPRITE_0
0x0093 -> ADDRESS_SPRITE_1
0x0094 -> BIRD_HIGH
0x0095 -> BIRD_VPOS
0x0096 -> BIRD_HPOS
0x0097 -> BIRD_HEAD_OFFSET
0x0098 -> PATTERN_INDEX
0x0099 -> FOLIAGE_SEED
0x009a -> OBSTACLE_SEED
0x009b -> BRANCH_SEED
0x009c -> OB_0
0x009e -> OB_1
0x00a0 -> OB_0_BRANCH
0x00a1 -> OB_1_BRANCH
0x00a2 -> OB_0_HPOS
0x00a3 -> OB_1_HPOS
0x00a4 -> OB_0_SPEED
0x00a5 -> OB_1_SPEED
0x00a6 -> FOREST_MID_0
0x00a7 -> FOREST_MID_1
0x00a8 -> FOREST_MID_2
0x00a9 -> SPLASH_COLOR
0x00aa -> SCORE
0x00ab -> HISCORE
0x00b0 -> SWAMP_COLOR
0x00b2 -> SWAMP_BACKGROUND
0x00c0 -> DISPLAY_SCANLINES
0x00d0 -> FOLIAGE_COLOR
0x00d2 -> FOREST_BACKGROUND
0x00e0 -> FOREST_COLOR
0x00e4 -> 135.CLOCK_COUNTS_PER_SCANLINE
0x00f2 -> _PAGE_CHECK
0x00f6 -> HISCORE_COLOR
0x00fe -> PLAY_STATE_DROWN
0x00ff -> PLAY_STATE_COLLISION
0x0280 -> SWCHA
0x0281 -> SWACNT
0x0282 -> SWCHB
0x0283 -> SWBCNT
0x0284 -> INTIM
0x0285 -> TIMINT
0x0286 -> TIM64T
0x0287 -> T1024T

Write Symbols
-------------
0x0000 -> VSYNC
0x0001 -> VBLANK
0x0002 -> WSYNC
0x0003 -> RSYNC
0x0004 -> NUSIZ0
0x0005 -> NUSIZ1
0x0006 -> COLUP0
0x0007 -> COLUP1
0x0008 -> COLUPF
0x0009 -> COLUBK
0x000a -> CTRLPF
0x000b -> REFP0
0x000c -> REFP1
0x000d -> PF0
0x000e -> PF1
0x000f -> PF2
0x0010 -> RESP0
0x0011 -> RESP1
0x0012 -> RESM0
0x0013 -> RESM1
0x0014 -> RESBL
0x0015 -> AUDC0
0x0016 -> AUDC1
0x0017 -> AUDF0
0x0018 -> AUDF1
0x0019 -> AUDV0
0x001a -> AUDV1
0x001b -> GRP0
0x001c -> GRP1
0x001d -> ENAM0
0x001e -> ENAM1
0x001f -> ENABL
0x0020 -> HMP0
0x0021 -> HMP1
0x0022 -> HMM0
0x0023 -> HMM1
0x0024 -> HMBL
0x0025 -> VDELP0
0x0026 -> VDELP1
0x0027 -> VDELBL
0x0028 -> RESMP0
0x0029 -> RESMP1
0x002a -> HMOVE
0x002b -> HMCLR
0x002c -> CXCLR
0x002e -> 129.CYCLES
0x002f -> BIRD_VPOS_INIT
0x0030 -> BRANCH_WIDTH
0x0032 -> 132.CYCLES
0x003c -> 123.CYCLES
0x0080 -> __MULTI_COUNT_STATE
0x0081 -> __STATE_INPT4
0x0082 -> __STATE_SWCHB
0x0083 -> __SFX_NEW_EVENT
0x0084 -> __SFX_QUEUE_EVENT
0x0085 -> __SFX_SUB_FRAMES
0x0086 -> _localA
0x0087 -> _localB
0x0088 -> _localC
0x0089 -> _localD
0x008a -> _localE
0x008b -> _localF
0x008c -> _localG
0x008d -> PLAY_STATE
0x008e -> SELECTED_HEAD
0x008f -> FLIGHT_PATTERN
0x0091 -> ADDRESS_SPRITE_0
0x0093 -> ADDRESS_SPRITE_1
0x0094 -> BIRD_HIGH
0x0095 -> BIRD_VPOS
0x0096 -> BIRD_HPOS
0x0097 -> BIRD_HEAD_OFFSET
0x0098 -> PATTERN_INDEX
0x0099 -> FOLIAGE_SEED
0x009a -> OBSTACLE_SEED
0x009b -> BRANCH_SEED
0x009c -> OB_0
0x009e -> OB_1
0x00a0 -> OB_0_BRANCH
0x00a1 -> OB_1_BRANCH
0x00a2 -> OB_0_HPOS
0x00a3 -> OB_1_HPOS
0x00a4 -> OB_0_SPEED
0x00a5 -> OB_1_SPEED
0x00a6 -> FOREST_MID_0
0x00a7 -> FOREST_MID_1
0x00a8 -> FOREST_MID_2
0x00a9 -> SPLASH_COLOR
0x00aa -> SCORE
0x00ab -> HISCORE
0x00b0 -> SWAMP_COLOR
0x00b2 -> SWAMP_BACKGROUND
0x00c0 -> DISPLAY_SCANLINES
0x00d0 -> FOLIAGE_COLOR
0x00d2 -> FOREST_BACKGROUND
0x00e0 -> FOREST_COLOR
0x00e4 -> 135.CLOCK_COUNTS_PER_SCANLINE
0x00f2 -> _PAGE_CHECK
0x00f6 -> HISCORE_COLOR
0x00fe -> PLAY_STATE_DROWN
0x00ff -> PLAY_STATE_COLLISION
0x0280 -> SWCHA
0x0281 -> SWACNT
0x0282 -> SWCHB
0x0283 -> SWBCNT
0x0284 -> INTIM
0x0285 -> TIMINT
0x0294 -> TIM1T
0x0295 -> TIM8T
0x0296 -> TIM64T
0x0297 -> T1024T
`
