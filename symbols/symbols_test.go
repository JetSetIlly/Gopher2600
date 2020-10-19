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
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/symbols"
	"github.com/jetsetilly/gopher2600/test"
)

func TestDefaultSymbols(t *testing.T) {
	syms := symbols.NewSymbols()
	tw := &test.Writer{}

	syms.ListSymbols(tw)

	if !tw.Compare(expectedDefaultSymbols) {
		t.Errorf("default symbols list is wrong")
	}
}

func TestFlappySymbols(t *testing.T) {
	// make a dummy cartridge with the minimum amount of information required
	// for ReadSymbolsFile() to work
	cart := cartridge.NewCartridge(nil)
	cart.Filename = "testdata/flappy.bin"

	syms, err := symbols.ReadSymbolsFile(cart)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}

	tw := &test.Writer{}

	syms.ListSymbols(tw)

	if !tw.Compare(expectedFlappySymbols) {
		t.Errorf("flappy symbols list is wrong")
	}
}

const expectedDefaultSymbols = `Labels
---------

Read Symbols
-----------
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
------------
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
---------
0x0000 -> .FREE_BYTES
0x0002 -> .CYCLES
0x0003 -> .CLOCK_COUNTS_PER_CYCLE
0x0005 -> .SCANLINES
0x0006 -> .KERNEL_TIMER_SET_IN_CYCLES
0x0007 -> .KERNEL_WAIT_LOOP
0x000b -> .TIMER_VAL
0x000c -> .CYCLES
0x000e -> .SCANLINES
0x0086 -> .PF0
0x0087 -> .PF1
0x0088 -> .PF2
0x0089 -> .MISSILE_1_NUSIZ
0x008a -> .PLAYER_0_SPRITE
0x008b -> .PLAYER_1_SPRITE
0x00e4 -> .CLOCK_COUNTS_PER_SCANLINE
0x1285 -> .CLEAR_STACK
0x128d -> .vsync
0x128f -> .VSLP1
0x1296 -> .vblank
0x12a5 -> .done
0x12ab -> .vblank_loop
0x12b4 -> .visible_loop
0x12b9 -> .overscan
0x12c1 -> .overscan_loop
0x12c9 -> .end_title_screen
0x12d2 -> .overscan_kernel
0x12da -> .overscan_loop
0x133c -> .done
0x134b -> .done_hiscore
0x1350 -> .VSLP1
0x135e -> .coarse_div
0x1362 -> .done_coarse_div
0x1375 -> .coarse_div
0x1379 -> .done_coarse_div
0x13bd -> .VSLP1
0x13de -> .far_jmp_collision
0x13e1 -> .far_jmp_drown
0x13e4 -> .far_jmp_approach
0x13e7 -> .far_jmp_play
0x13f2 -> .done
0x1408 -> .no_store_index
0x141a -> .ready_state_triage
0x1428 -> .update_foliage
0x1431 -> .rotate_forest
0x143d -> .jump_tree
0x1444 -> .cont_forest
0x144d -> .carry_tree
0x1453 -> .forest_done
0x1455 -> .prepare_display
0x145b -> .display_empty
0x1466 -> .display_ready_logo
0x147d -> .coarse_div
0x1481 -> .done_coarse_div
0x1492 -> .coarse_div
0x1496 -> .done_coarse_div
0x14b2 -> .update_foliage
0x14bb -> .foliage_updated
0x14c0 -> .update_bird
0x14ca -> .use_wings_up
0x14d1 -> .use_wings_flat
0x14d8 -> .use_wings_down
0x14dc -> .wings_updated
0x14ed -> .update_pattern_idx
0x14f9 -> .store_index
0x1505 -> .enter_drowning_state
0x1525 -> .prepare_display
0x152a -> .coarse_div
0x152e -> .done_coarse_div
0x1542 -> .coarse_div
0x1546 -> .done_coarse_div
0x1562 -> .update_foliage
0x156b -> .foliage_updated
0x1570 -> .update_bird
0x157b -> .drowning_end
0x157e -> .prepare_display
0x1583 -> .coarse_div
0x1587 -> .done_coarse_div
0x159b -> .coarse_div
0x159f -> .done_coarse_div
0x15b0 -> .coarse_div
0x15b4 -> .done_coarse_div
0x15cb -> .coarse_div
0x15cf -> .done_coarse_div
0x15de -> .show_obstacle_1
0x15e3 -> .coarse_div
0x15e7 -> .done_coarse_div
0x15f3 -> .flipped_obstacles
0x161e -> .hpos_done
0x162c -> .done_completion_test
0x1639 -> .far_jmp_sprite
0x1645 -> .rotate_forest
0x1651 -> .jump_tree
0x1658 -> .cont_forest
0x1661 -> .carry_tree
0x1667 -> .forest_done
0x1689 -> .reset_obstacle_0
0x1696 -> .reset_obstacle_1
0x16aa -> .bird_collision
0x16b8 -> .no_store_index
0x16c4 -> .done_vblank_collisions
0x16cf -> .done
0x16ec -> .flip_sprite_use_flat
0x16f0 -> .flip_sprite_end
0x1700 -> .use_wings_up_sprite
0x1707 -> .use_glide_sprite
0x170b -> .sprite_set
0x1716 -> .begin_drowning
0x1739 -> .limit_height
0x173b -> .update_pattern_idx
0x1747 -> .store_index
0x1749 -> .fly_end
0x174e -> .coarse_div
0x1752 -> .done_coarse_div
0x1766 -> .coarse_div
0x176a -> .done_coarse_div
0x177f -> .fine_move_done
0x178a -> .fine_move_done
0x1791 -> .coarse_div
0x1795 -> .done_coarse_div
0x17a7 -> .coarse_div
0x17ab -> .done_coarse_div
0x17b7 -> .scoring_check
0x17c8 -> .score_obstacle
0x17d4 -> .end_scoring
0x17ec -> .vblank_loop
0x17f7 -> .next_foliage
0x180e -> .set_trunk
0x1814 -> .new_foliage
0x1816 -> .cont_foliage
0x183c -> .precalc_forest_static
0x1851 -> .end_forest_precalc
0x187d -> .set_missile_sprites
0x188d -> .precalc_missile_size
0x1895 -> .done_precalc_missile_size
0x1897 -> .precalc_missile_sprites
0x18a2 -> .set_player_sprites
0x18ae -> .precalc_players_sprites
0x18c3 -> .done_precalc_players
0x18cb -> .next_scanline
0x18d6 -> .draw_swamp
0x1923 -> .prep_score
0x197e -> .prep_hiscore
0x19cc -> .tens_digits
0x19da -> .scoring_loop
0x19e4 -> .next_scanline
0x1a04 -> .done
0x1a0f -> .swap_heads
0x1a16 -> .done_head_check
0x1a38 -> .set_width_for_ready
0x1a40 -> .done_set_width
0x1a4a -> .next_obstacle
0x1a54 -> .next_branch
0x1a7c -> .done_drowning_compensation
0x1a82 -> .sfx_new_event
0x1a8d -> .sfx_queue_event
0x1ac9 -> .sfx_cont
0x1ad7 -> .sfx_done
0x1adb -> .is_positive
0x1ae5 -> .positive_reset
0x1aea -> .is_negative
0x1af1 -> .negative_reset
0x1af3 -> .store
0x1af5 -> .overscan_loop

Read Symbols
-----------
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
0x128d -> title_screen
0x12df -> game_state_init
0x1332 -> game_restart
0x13bb -> game_vsync
0x13c4 -> game_vblank
0x13ea -> game_vblank_ready
0x14a5 -> game_vblank_death_collision
0x1555 -> game_vblank_death_drown
0x160a -> game_vblank_approach
0x162c -> game_vblank_main_triage
0x163c -> game_vblank_foliage
0x166c -> game_vblank_collisions
0x16c7 -> game_vblank_sprite
0x1749 -> game_vblank_position_sprites
0x17d4 -> game_vblank_end
0x17f7 -> foliage
0x1823 -> game_play_area_prepare
0x186d -> game_play_area
0x18d6 -> swamp
0x1901 -> display_score
0x19ea -> game_overscan
0x1afd -> initialisation

Write Symbols
------------
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
0x002e -> OKAY_COLOR
0x002f -> BIRD_VPOS_INIT
0x0030 -> BRANCH_WIDTH
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
0x128d -> title_screen
0x12df -> game_state_init
0x1332 -> game_restart
0x13bb -> game_vsync
0x13c4 -> game_vblank
0x13ea -> game_vblank_ready
0x14a5 -> game_vblank_death_collision
0x1555 -> game_vblank_death_drown
0x160a -> game_vblank_approach
0x162c -> game_vblank_main_triage
0x163c -> game_vblank_foliage
0x166c -> game_vblank_collisions
0x16c7 -> game_vblank_sprite
0x1749 -> game_vblank_position_sprites
0x17d4 -> game_vblank_end
0x17f7 -> foliage
0x1823 -> game_play_area_prepare
0x186d -> game_play_area
0x18d6 -> swamp
0x1901 -> display_score
0x19ea -> game_overscan
0x1afd -> initialisation
`
