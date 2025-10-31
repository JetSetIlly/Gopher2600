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

// Package klaus2m5 contains the various 6502 functional tests as
// created/maintained by Klaus Dormann.
//
// https://github.com/Klaus2m5/6502_65C02_functional_tests
//
// The tests were assembled with as65 assembler which is available for download
// at the above URL. In all cases the assembler was executed in the following
// manner:
//
// as65 -pmnu <test file>.a65
//
// The compiled binaries of the configured tests are placed in individual
// directories described below.
//
// # functional_test
//
// This directory contains the 6502_functional_test.a65 file with the vectors
// test disabled.
//
// line 88: ROM_vectors = 0
//
// # decimal_mode
//
// This directory contains the 6502_decimal_test.a65 file. changed so that the
// sign and zero flags are tested.
//
// line 31: check_n = 1
// line 33: check_z = 1
//
// The overflow flag check is currently disabled.
package klaus2m5
