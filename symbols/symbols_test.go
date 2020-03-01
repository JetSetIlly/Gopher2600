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

package symbols_test

import (
	"gopher2600/symbols"
	"gopher2600/test"
	"testing"
)

func TestDefaultSymbols(t *testing.T) {
	syms, err := symbols.ReadSymbolsFile("")
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}

	tw := &test.Writer{}

	syms.ListSymbols(tw)

	if !tw.Compare(expectedDefaultSymbols) {
		t.Errorf("default symbols list is wrong")
	}
}

func TestFlappySymbols(t *testing.T) {
	syms, err := symbols.ReadSymbolsFile("testdata/flappy.sym")
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}

	tw := &test.Writer{}

	syms.ListSymbols(tw)

	if !tw.Compare(expectedFlappySymbols) {
		t.Errorf("flappy symbols list is wrong")
	}
}

const expectedDefaultSymbols = `Locations
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

const expectedFlappySymbols = `Locations
---------
0x0000 -> .FREE_BYTES
0x0003 -> .CLOCK_COUNTS_PER_CYCLE
0x0006 -> .KERNEL_TIMER_SET_IN_CYCLES
0x0007 -> .KERNEL_WAIT_LOOP
0x001e -> .SCANLINES
0x0023 -> .TIMER_VAL
0x0025 -> .SCANLINES
0x002b -> .TIMER_VAL
0x002e -> .CYCLES
0x0032 -> .CYCLES
0x003c -> .CYCLES
0x0040 -> .CYCLES
0x004c -> .CYCLES_PER_SCANLINE
0x0086 -> .YSTATE
0x0087 -> .PF1
0x0088 -> .MISSILE_1_SET
0x0089 -> .MISSILE_1_NUSIZ
0x008a -> .PLAYER_0_SPRITE
0x008b -> .PLAYER_1_SPRITE
0x00e4 -> .CLOCK_COUNTS_PER_SCANLINE
0xf282 -> .vsync
0xf284 -> .VSLP1
0xf28b -> .vblank
0xf29a -> .done
0xf2a0 -> .vblank_loop
0xf2a9 -> .visible_loop
0xf2ae -> .overscan
0xf2b6 -> .overscan_loop
0xf2be -> .end_title_screen
0xf2c7 -> .overscan_kernel
0xf2cf -> .overscan_loop
0xf331 -> .done
0xf340 -> .done_hiscore
0xf345 -> .VSLP1
0xf353 -> .coarse_div
0xf357 -> .done_coarse_div
0xf36a -> .coarse_div
0xf36e -> .done_coarse_div
0xf3b2 -> .VSLP1
0xf3d3 -> .far_jmp_collision
0xf3d6 -> .far_jmp_drown
0xf3d9 -> .far_jmp_approach
0xf3dc -> .far_jmp_play
0xf3e7 -> .done
0xf3fd -> .no_store_index
0xf40f -> .ready_state_triage
0xf41d -> .update_foliage
0xf426 -> .rotate_forest
0xf432 -> .jump_tree
0xf439 -> .cont_forest
0xf442 -> .carry_tree
0xf448 -> .forest_done
0xf44a -> .prepare_display
0xf450 -> .display_empty
0xf45b -> .display_ready_logo
0xf472 -> .coarse_div
0xf476 -> .done_coarse_div
0xf487 -> .coarse_div
0xf48b -> .done_coarse_div
0xf4a7 -> .update_foliage
0xf4b0 -> .foliage_updated
0xf4b5 -> .update_bird
0xf4bf -> .use_wings_up
0xf4c6 -> .use_wings_flat
0xf4cd -> .use_wings_down
0xf4d1 -> .wings_updated
0xf4e2 -> .update_pattern_idx
0xf4ee -> .store_index
0xf4fa -> .enter_drowning_state
0xf51a -> .prepare_display
0xf51f -> .coarse_div
0xf523 -> .done_coarse_div
0xf537 -> .coarse_div
0xf53b -> .done_coarse_div
0xf557 -> .update_foliage
0xf560 -> .foliage_updated
0xf565 -> .update_bird
0xf570 -> .drowning_end
0xf573 -> .prepare_display
0xf578 -> .coarse_div
0xf57c -> .done_coarse_div
0xf590 -> .coarse_div
0xf594 -> .done_coarse_div
0xf5a5 -> .coarse_div
0xf5a9 -> .done_coarse_div
0xf5c0 -> .coarse_div
0xf5c4 -> .done_coarse_div
0xf5d3 -> .show_obstacle_1
0xf5d8 -> .coarse_div
0xf5dc -> .done_coarse_div
0xf5e8 -> .flipped_obstacles
0xf613 -> .hpos_done
0xf621 -> .done_completion_test
0xf62e -> .far_jmp_sprite
0xf63a -> .rotate_forest
0xf646 -> .jump_tree
0xf64d -> .cont_forest
0xf656 -> .carry_tree
0xf65c -> .forest_done
0xf67e -> .reset_obstacle_0
0xf68b -> .reset_obstacle_1
0xf69f -> .bird_collision
0xf6ad -> .no_store_index
0xf6b9 -> .done_vblank_collisions
0xf6c4 -> .done
0xf6e1 -> .flip_sprite_use_flat
0xf6e5 -> .flip_sprite_end
0xf6f5 -> .use_wings_up_sprite
0xf6fc -> .use_glide_sprite
0xf700 -> .sprite_set
0xf70b -> .begin_drowning
0xf72e -> .limit_height
0xf730 -> .update_pattern_idx
0xf73c -> .store_index
0xf73e -> .fly_end
0xf743 -> .coarse_div
0xf747 -> .done_coarse_div
0xf75b -> .coarse_div
0xf75f -> .done_coarse_div
0xf774 -> .fine_move_done
0xf77f -> .fine_move_done
0xf786 -> .coarse_div
0xf78a -> .done_coarse_div
0xf79c -> .coarse_div
0xf7a0 -> .done_coarse_div
0xf7ac -> .scoring_check
0xf7bd -> .score_obstacle
0xf7c9 -> .end_scoring
0xf7e1 -> .vblank_loop
0xf7ec -> .next_foliage
0xf803 -> .set_trunk
0xf809 -> .new_foliage
0xf80b -> .cont_foliage
0xf831 -> .precalc_forest_static
0xf846 -> .end_forest_precalc
0xf872 -> .set_missile_sprites
0xf882 -> .precalc_missile_size
0xf88a -> .done_precalc_missile_size
0xf88c -> .precalc_missile_sprites
0xf897 -> .set_player_sprites
0xf8a3 -> .precalc_players_sprites
0xf8b8 -> .done_precalc_players
0xf8c0 -> .next_scanline
0xf8cb -> .draw_swamp
0xf918 -> .prep_score
0xf973 -> .prep_hiscore
0xf9c1 -> .tens_digits
0xf9cf -> .scoring_loop
0xf9d9 -> .next_scanline
0xf9f9 -> .done
0xfa04 -> .swap_heads
0xfa0b -> .done_head_check
0xfa2d -> .set_width_for_ready
0xfa35 -> .done_set_width
0xfa3f -> .next_obstacle
0xfa49 -> .next_branch
0xfa71 -> .done_drowning_compensation
0xfa77 -> .sfx_new_event
0xfa82 -> .sfx_queue_event
0xfabe -> .sfx_cont
0xfacc -> .sfx_done
0xfad0 -> .is_positive
0xfada -> .positive_reset
0xfadf -> .is_negative
0xfae6 -> .negative_reset
0xfae8 -> .store
0xfaea -> .overscan_loop

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
0x0020 -> OBSTACLE_WIDTH
0x0021 -> HMP1
0x0022 -> HMM0
0x0023 -> HMM1
0x0024 -> CTRLPF_FOLIAGE
0x0025 -> VDELP0
0x0026 -> VDELP1
0x0027 -> VDELBL
0x0028 -> RESMP0
0x0029 -> RESMP1
0x002a -> HMOVE
0x002b -> HMCLR
0x002c -> CXCLR
0x002e -> OKAY_COLOR
0x0030 -> BRANCH_WIDTH
0x0069 -> VERSION_VCS
0x006a -> VERSION_MACRO
0x006c -> BIRD_VPOS_INIT
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
0x0093 -> BIRD_HIGH
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
0x2e20 -> _MSG_MARKER
0xf000 -> DATA_SEGMENT
0xf008 -> TEXT_OK
0xf010 -> TEXT_QMARK
0xf018 -> WINGS
0xf020 -> WINGS_FLAT
0xf028 -> WINGS_DOWN
0xf030 -> HEADS
0xf038 -> HEAD_BOY_A
0xf040 -> HEAD_GIRL_B
0xf048 -> HEAD_BOY_B
0xf050 -> HEADS_TABLE
0xf054 -> _SPLASH
0xf056 -> SPLASH
0xf05e -> DIGIT_0
0xf063 -> DIGIT_1
0xf068 -> DIGIT_2
0xf06d -> DIGIT_3
0xf072 -> DIGIT_4
0xf077 -> DIGIT_5
0xf07c -> DIGIT_6
0xf081 -> DIGIT_7
0xf086 -> DIGIT_8
0xf08b -> DIGIT_9
0xf090 -> DIGIT_TABLE
0xf09a -> FOLIAGE
0xf0b8 -> FOREST_MID_0_INIT
0xf0b9 -> FOREST_MID_1_INIT
0xf0ba -> FOREST_MID_2_INIT
0xf0bb -> FOREST_STATIC_0
0xf0bc -> FOREST_STATIC_1
0xf0bd -> FOREST_STATIC_2
0xf100 -> SET_OBSTACLE_TABLE
0xf137 -> FINE_POS_TABLE
0xf204 -> OBSTACLES
0xf20b -> BRANCHES
0xf212 -> EASY_FLIGHT_PATTERN
0xf228 -> __FINE_POS_TABLE
0xf237 -> SFX_TABLE
0xf23d -> SFX_FLAP
0xf243 -> SFX_COLLISION
0xf255 -> SFX_SPLASH
0xf27f -> setup
0xf282 -> title_screen
0xf2d4 -> game_state_init
0xf327 -> game_restart
0xf3b0 -> game_vsync
0xf3b9 -> game_vblank
0xf3df -> game_vblank_ready
0xf49a -> game_vblank_death_collision
0xf54a -> game_vblank_death_drown
0xf5ff -> game_vblank_approach
0xf621 -> game_vblank_main_triage
0xf631 -> game_vblank_foliage
0xf661 -> game_vblank_collisions
0xf6bc -> game_vblank_sprite
0xf73e -> game_vblank_position_sprites
0xf7c9 -> game_vblank_end
0xf7ec -> foliage
0xf818 -> game_play_area_prepare
0xf862 -> game_play_area
0xf8cb -> swamp
0xf8f6 -> display_score
0xf9df -> game_overscan
0xfaf2 -> initialisation

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
0x0030 -> BRANCH_WIDTH
0x0069 -> VERSION_VCS
0x006a -> VERSION_MACRO
0x006c -> BIRD_VPOS_INIT
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
0x0093 -> BIRD_HIGH
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
0x2e20 -> _MSG_MARKER
0xf000 -> DATA_SEGMENT
0xf008 -> TEXT_OK
0xf010 -> TEXT_QMARK
0xf018 -> WINGS
0xf020 -> WINGS_FLAT
0xf028 -> WINGS_DOWN
0xf030 -> HEADS
0xf038 -> HEAD_BOY_A
0xf040 -> HEAD_GIRL_B
0xf048 -> HEAD_BOY_B
0xf050 -> HEADS_TABLE
0xf054 -> _SPLASH
0xf056 -> SPLASH
0xf05e -> DIGIT_0
0xf063 -> DIGIT_1
0xf068 -> DIGIT_2
0xf06d -> DIGIT_3
0xf072 -> DIGIT_4
0xf077 -> DIGIT_5
0xf07c -> DIGIT_6
0xf081 -> DIGIT_7
0xf086 -> DIGIT_8
0xf08b -> DIGIT_9
0xf090 -> DIGIT_TABLE
0xf09a -> FOLIAGE
0xf0b8 -> FOREST_MID_0_INIT
0xf0b9 -> FOREST_MID_1_INIT
0xf0ba -> FOREST_MID_2_INIT
0xf0bb -> FOREST_STATIC_0
0xf0bc -> FOREST_STATIC_1
0xf0bd -> FOREST_STATIC_2
0xf100 -> SET_OBSTACLE_TABLE
0xf137 -> FINE_POS_TABLE
0xf204 -> OBSTACLES
0xf20b -> BRANCHES
0xf212 -> EASY_FLIGHT_PATTERN
0xf228 -> __FINE_POS_TABLE
0xf237 -> SFX_TABLE
0xf23d -> SFX_FLAP
0xf243 -> SFX_COLLISION
0xf255 -> SFX_SPLASH
0xf27f -> setup
0xf282 -> title_screen
0xf2d4 -> game_state_init
0xf327 -> game_restart
0xf3b0 -> game_vsync
0xf3b9 -> game_vblank
0xf3df -> game_vblank_ready
0xf49a -> game_vblank_death_collision
0xf54a -> game_vblank_death_drown
0xf5ff -> game_vblank_approach
0xf621 -> game_vblank_main_triage
0xf631 -> game_vblank_foliage
0xf661 -> game_vblank_collisions
0xf6bc -> game_vblank_sprite
0xf73e -> game_vblank_position_sprites
0xf7c9 -> game_vblank_end
0xf7ec -> foliage
0xf818 -> game_play_area_prepare
0xf862 -> game_play_area
0xf8cb -> swamp
0xf8f6 -> display_score
0xf9df -> game_overscan
0xfaf2 -> initialisation
`
